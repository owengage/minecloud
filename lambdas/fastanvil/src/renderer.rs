use std::io::Cursor;

use fastanvil::{parse_region, Region, RegionBlockDrawer, RegionMap, RenderedPalette};
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

    pub fn render(&self, region: &[u8]) -> Vec<u8> {
        let cursor = Cursor::new(region);
        let region = Region::new(cursor);

        let map = RegionMap::new(0, 0, [0, 0, 0, 0]);
        let mut drawer = RegionBlockDrawer::new(map, &self.pal);

        parse_region(region, &mut drawer).unwrap_or_default(); // TODO handle some of the errors here

        let region = drawer.map;
        let region_len: usize = 32 * 16;

        let mut img = image::ImageBuffer::new(region_len as u32, region_len as u32);

        for xc in 0..32 {
            for zc in 0..32 {
                let chunk = region.chunk(xc, zc);
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
