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
    let client = S3Client::new(Region::EuWest2);
    let r = renderer::TileRenderer::new();

    for record in e.records {
        let bucket = record.s3.bucket.name.unwrap();
        let key = record.s3.object.key.unwrap();

        let obj = client.get_object(GetObjectRequest {
            bucket: bucket.clone(),
            key: key.clone(),
            ..Default::default()
        });

        if !key.ends_with(".mca") {
            simple_error::bail!("not a region");
        }

        let mut buf = vec![];
        obj.await
            .unwrap()
            .body
            .unwrap()
            .into_async_read()
            .read_to_end(&mut buf)
            .await
            .unwrap();

        info!("object size {}", buf.len());
        let img = r.render(buf.as_slice());

        info!("image bytes size {}", img.len());
        client
            .put_object(PutObjectRequest {
                bucket: bucket,
                key: "tiles/test.png".to_string(),
                body: Some(ByteStream::from(img)),
                ..Default::default()
            })
            .await
            .unwrap();

        // DONE: Download region from S3.
        // DONE: Parse region into image.
        // TODO: Upload image to other bucket.
        // TODO: Make upload to owengage.com bucket
        // TODO: Set up proper execution role. Need one to be able to write to
        // owengage.com.
        // TODO: Base name on key that triggered it.
        // TODO: Set up trigger.
        // TODO: Test world upload.
        // TODO: Make website be able to see it...
    }

    Ok(())
}
