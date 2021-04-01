use std::io::Cursor;

use fastanvil::{
    CCoord, Chunk, HeightMode, Palette, RCoord, Region, RegionBuffer, RegionMap, RenderedPalette,
    Rgba, TopShadeRenderer,
};
use image::{png::PngEncoder, ColorType};

use crate::palette::get_palette;

pub struct TileRenderer {
    pal: RenderedPalette,
}

impl TileRenderer {
    pub fn new() -> Self {
        Self {
            pal: get_palette().unwrap(),
        }
    }

    pub fn render(&self, region: &[u8], north_region: Option<&[u8]>) -> Vec<u8> {
        let region = RegionBuffer::new(Cursor::new(region));
        let north_region = north_region.map(|buf| RegionBuffer::new(Cursor::new(buf)));
        let north_region = north_region.as_ref().map(|r| r as &dyn Region);

        let renderer = TopShadeRenderer::new(&self.pal, HeightMode::Calculate);

        let region = render_region(&region, north_region, renderer); // TODO handle some of the errors here

        let region_len: usize = 32 * 16;

        let mut img = image::ImageBuffer::new(region_len as u32, region_len as u32);

        for xc in 0..32 {
            for zc in 0..32 {
                let chunk = region.chunk(CCoord(xc), CCoord(zc));
                let xcp = xc as isize;
                let zcp = zc as isize;

                for z in 0..16 {
                    for x in 0..16 {
                        let pixel = chunk[z * 16 + x];
                        let x = xcp * 16 + x as isize;
                        let z = zcp * 16 + z as isize;
                        img.put_pixel(x as u32, z as u32, image::Rgba(pixel))
                    }
                }
            }
        }

        let mut png = Vec::<u8>::new();
        let enc = PngEncoder::new(&mut png);
        enc.encode(img.as_raw(), img.width(), img.height(), ColorType::Rgba8)
            .unwrap();

        png
    }
}

// TODO: Reconcile with the similar method in fastanvil/src/render.rs
pub fn render_region<P: Palette>(
    region: &dyn Region,
    north_region: Option<&dyn Region>,
    renderer: TopShadeRenderer<P>,
) -> RegionMap<Rgba> {
    let mut map = RegionMap::new(RCoord(0), RCoord(0), [0u8; 4]);

    let mut cache: [Option<Box<dyn Chunk>>; 32] = Default::default();

    // Cache the last row of chunks from the above region to allow top-shading
    // on region boundaries.
    if let Some(north_region) = north_region {
        for x in 0..32 {
            cache[x] = north_region.chunk(CCoord(x as isize), CCoord(31));
        }
    }

    for z in 0isize..32 {
        for x in 0isize..32 {
            let (x, z) = (CCoord(x), CCoord(z));
            let data = map.chunk_mut(x, z);

            let chunk_data = region.chunk(x, z).map(|chunk| {
                // Get the chunk at the same x coordinate from the cache. This
                // should be the chunk that is directly above the current. We
                // know this because once we have processed this chunk we put it
                // in the cache in the same place. So the next time we get the
                // current one will be when we're processing directly below us.
                //
                // Thanks to the default None value this works fine for the
                // first row or for any missing chunks.
                let north = cache[x.0 as usize].as_ref().map(|c| &**c);
                let res = renderer.render(&*chunk, north);
                cache[x.0 as usize] = Some(chunk);
                res
            });

            chunk_data.map(|d| {
                data[..].clone_from_slice(&d);
            });
        }
    }

    map
}
