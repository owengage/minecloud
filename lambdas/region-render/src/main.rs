use lambda_runtime::{handler_fn, Context};
use log::{self, error, info, warn, LevelFilter};
use rusoto_core::{ByteStream, Region};
use rusoto_s3::{GetObjectRequest, PutObjectRequest, S3Client, S3};
use tokio::io::AsyncReadExt;

use aws_lambda_events::event::s3::S3Event;

type Error = Box<dyn std::error::Error + Sync + Send + 'static>;

mod palette;
mod renderer;

#[tokio::main]
async fn main() -> Result<(), Error> {
    simple_logger::SimpleLogger::new()
        .with_level(LevelFilter::Info)
        .init()
        .unwrap();

    lambda_runtime::run(handler_fn(handler)).await?;
    Ok(())
}

#[derive(Debug)]
struct TileDetails<'a> {
    world: &'a str,
    dimension: &'a str,
    x: isize,
    z: isize,
}

async fn handler(e: S3Event, _: Context) -> Result<(), Error> {
    let dest_bucket = "owengage.com";

    let client = S3Client::new(Region::EuWest2);
    let r = renderer::TileRenderer::new();

    for record in e.records {
        let bucket = record.s3.bucket.name.expect("could not get bucket name");
        let key = record.s3.object.key.expect("could not get s3 key");

        let re = regex::Regex::new(
            r"^worlds/(?P<world>[\-0-9A-Za-z_]+)/(?P<dimension>[\-0-9A-Za-z_]+)/r\.(?P<x>[\-0-9]+)\.(?P<z>[\-0-9]+)\.mca$"
        )
        .expect("could not compile regex");

        let details = match re.captures(&key) {
            Some(cap) => extract_details(cap),
            None => {
                warn!("key not of expected format, skipping: {}", key);
                return Ok(());
            }
        };

        info!("tile to process: {:?}", details);

        let tile_key = format!(
            "anvil-tiles/{}/{}/r.{}.{}.png",
            details.world, details.dimension, details.x, details.z,
        );

        let region_obj = client.get_object(GetObjectRequest {
            bucket: bucket.clone(),
            key: key.clone(),
            ..Default::default()
        });

        let mut region = vec![];
        region_obj
            .await
            .unwrap()
            .body
            .unwrap()
            .into_async_read()
            .read_to_end(&mut region)
            .await
            .expect("could not download region");

        let north_region_obj = client.get_object(GetObjectRequest {
            bucket: bucket.clone(),
            key: format!(
                "worlds/{}/{}/r.{}.{}.mca",
                details.world,
                details.dimension,
                details.x,
                details.z - 1
            ),
            ..Default::default()
        });

        // TODO: LET IT FAIL.
        let mut north_region = vec![];
        let north_req = north_region_obj.await;

        match north_req {
            Ok(north_req) => {
                north_req
                    .body
                    .expect("could not read north region body")
                    .into_async_read()
                    .read_to_end(&mut north_region)
                    .await
                    .expect("count not download north region");
            }
            Err(err) => {
                info!(
                    "could get north region (this is fine if there is no northern region): {:?}",
                    err
                );
            }
        }

        let img = if north_region.is_empty() {
            r.render(&region, None)
        } else {
            r.render(&region, Some(&north_region))
        };

        client
            .put_object(PutObjectRequest {
                bucket: dest_bucket.to_string(),
                key: tile_key,
                body: Some(ByteStream::from(img)),
                ..Default::default()
            })
            .await
            .expect("could not put region image");
    }

    Ok(())
}

fn extract_details(cap: regex::Captures) -> TileDetails {
    TileDetails {
        world: cap.name("world").unwrap().as_str(),
        dimension: cap.name("dimension").unwrap().as_str(),
        x: cap.name("x").unwrap().as_str().parse().unwrap(),
        z: cap.name("z").unwrap().as_str().parse().unwrap(),
    }
}
