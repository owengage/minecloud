use lambda_runtime::{error::HandlerError, lambda, Context};
use log::{self, info, LevelFilter};
use rusoto_core::{ByteStream, Region};
use rusoto_s3::{GetObjectRequest, PutObjectRequest, S3Client, S3};
use tokio::io::AsyncReadExt;
use tokio::runtime::Runtime;

use aws_lambda_events::event::s3::S3Event;
type Error = HandlerError;

mod palette;
mod renderer;

fn main() -> Result<(), Error> {
    simple_logger::SimpleLogger::new()
        .with_level(LevelFilter::Info)
        .init()
        .unwrap();

    lambda!(handler_wrapper);
    Ok(())
}

fn handler_wrapper(e: S3Event, c: Context) -> Result<(), Error> {
    let mut rt = Runtime::new().unwrap();
    rt.block_on(handler(e, c))?;
    Ok(())
}

async fn handler(e: S3Event, _: Context) -> Result<(), Error> {
    let dest_bucket = "owengage.com";
    let dest_prefix = "anvil-tiles/";

    let client = S3Client::new(Region::EuWest2);
    let r = renderer::TileRenderer::new();

    for record in e.records {
        let bucket = record.s3.bucket.name.expect("could not get bucket name");
        let key = record.s3.object.key.expect("could not get s3 key");

        let re = regex::Regex::new(
            r"^worlds/(?P<key>[\-0-9A-Za-z_]+/region/r\.[\-0-9]+\.[\-0-9]+)\.mca$",
        )
        .expect("could not compile regex");

        let captures = match re.captures(&key) {
            Some(cap) => cap,
            None => return Ok(()), // might be the end/nether. Just ignore for now.
        };

        let tile_key = dest_prefix.to_string() + captures.name("key").unwrap().as_str() + ".png";

        let obj = client.get_object(GetObjectRequest {
            bucket: bucket.clone(),
            key: key.clone(),
            ..Default::default()
        });

        let mut buf = vec![];
        obj.await
            .unwrap()
            .body
            .unwrap()
            .into_async_read()
            .read_to_end(&mut buf)
            .await
            .expect("could not download region");

        let img = r.render(buf.as_slice());

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
