#!/usr/bin/env python3
"""Compose real app captures into a silent, captioned 1080x1920 announcement reel."""
import argparse
from bisect import bisect_right
from functools import lru_cache
import json
import math
from pathlib import Path
import zipfile

import imageio_ffmpeg
from PIL import Image, ImageDraw, ImageFont, ImageFilter

ROOT = Path(__file__).resolve().parents[1]
WIDTH, HEIGHT, FPS, DURATION = 1080, 1920, 30, 22
FOREST, SPROUT, GOLD, WHITE, MUTED = '#1B5E3B', '#A4D65E', '#D9A441', '#F6F5EC', '#B8C7BC'
SCENES = [
    (0, 3.2, None, "Your next idea", "needs room.", "A little space for your thoughts."),
    (3.2, 7.2, 'collect', "Collect your", "inspiration.", "Images, notes, and ideas. In one place."),
    (7.2, 11.6, 'draw', "Draw it.", "Make it yours.", "Pen, highlighter, and hold-to-straighten."),
    (11.6, 16, 'share', "Bring someone", "into the idea.", "Private workspaces. Sharing. Comments."),
    (16, 22, None, "Want to try", "mindflow?", "I'm looking for early testers."),
]


def ease(t):
    t = min(1, max(0, t))
    return t * t * (3 - 2 * t)


def pick_font(bold=False):
    candidates = ([('/System/Library/Fonts/Avenir Next.ttc', 0), ('/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf', 0)]
                  if bold else [('/System/Library/Fonts/SFNS.ttf', 0), ('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 0)])
    for path, index in candidates:
        if Path(path).exists():
            return path, index
    raise RuntimeError('Install DejaVu Sans or supply --font and --bold-font.')


class Reel:
    def __init__(self, capture, font=None, bold_font=None):
        self.capture = capture
        self.clips = {clip['name']: clip for clip in json.loads((capture / 'clips.json').read_text())}
        self.face = (font, 0) if font else pick_font()
        self.bold_face = (bold_font, 0) if bold_font else pick_font(True)
        self.icon = Image.open(ROOT / 'web/public/icon.png').convert('RGBA')
        self.background = self.make_background()

    @lru_cache(maxsize=24)
    def font(self, size, bold=False):
        path, index = self.bold_face if bold else self.face
        return ImageFont.truetype(path, size, index=index)

    def make_background(self):
        image = Image.new('RGB', (WIDTH, HEIGHT))
        draw = ImageDraw.Draw(image)
        for y in range(HEIGHT):
            t = y / HEIGHT
            draw.line((0, y, WIDTH, y), fill=(round(8+9*t), round(23+17*t), round(17+12*t)))
        for y in range(38, HEIGHT, 48):
            for x in range(34, WIDTH, 48):
                draw.ellipse((x,y,x+2,y+2), fill=(33,62,43))
        return image.convert('RGBA')

    def text(self, image, value, x, y, size, color=WHITE, bold=False, centered=False):
        draw = ImageDraw.Draw(image)
        font = self.font(size, bold)
        if centered:
            x = (WIDTH-draw.textlength(value, font=font))/2
        draw.text((x, y), value, font=font, fill=color, stroke_width=0)

    def brand(self, image):
        icon = self.icon.resize((72,60), Image.Resampling.LANCZOS)
        image.alpha_composite(icon, (78,143))
        self.text(image, 'mindflow', 169, 141, 43, bold=True)
        self.text(image, 'EARLY ACCESS', 770, 157, 21, color=SPROUT, bold=True)

    @lru_cache(maxsize=12)
    def screenshot(self, name):
        with Image.open(self.capture / 'frames' / name) as image:
            return image.convert('RGB').copy()

    def source(self, clip_name, progress):
        clip = self.clips[clip_name]
        target = min(1, max(0, progress)) * max(1, clip['frames'][-1]['time'])
        index = max(0, bisect_right([frame['time'] for frame in clip['frames']], target)-1)
        return self.screenshot(clip['frames'][index]['file'])

    def app_window(self, image, source, y=610, size=932, progress=0):
        # A restrained push-in keeps the actual UI legible without inventing interactions.
        zoomed = source.resize((round(size*(1+.018*progress)),)*2, Image.Resampling.LANCZOS)
        margin = (zoomed.width-size)//2
        zoomed = zoomed.crop((margin,margin,margin+size,margin+size)).convert('RGBA')
        mask=Image.new('L',(size,size),0)
        ImageDraw.Draw(mask).rounded_rectangle((0,0,size-1,size-1), radius=30,fill=255)
        shadow=Image.new('RGBA',(WIDTH,HEIGHT),0)
        ImageDraw.Draw(shadow).rounded_rectangle((68,y+12,68+size,y+size+20),radius=32,fill=(0,0,0,110))
        image.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(22)))
        image.paste(zoomed,(74,y),mask)
        ImageDraw.Draw(image).rounded_rectangle((73,y-1,74+size,y+size),radius=31,outline='#3A5540',width=2)
        # Presentation frame, outside the captured app.
        ImageDraw.Draw(image).rounded_rectangle((74,y-39,1006,y+22),radius=24,fill='#1E3328')
        ImageDraw.Draw(image).rectangle((74,y-15,1006,y+1),fill='#1E3328')
        for x,c in [(98,'#D9A441'),(119,'#A4D65E'),(140,'#4CAF50')]:
            ImageDraw.Draw(image).ellipse((x,y-23,x+9,y-14),fill=c)
        self.text(image,'mindflow / a little space for your thoughts',390,y-29,15,color=MUTED)

    def scene(self, t):
        index = min(len(SCENES)-1, bisect_right([scene[0] for scene in SCENES], t)-1)
        start,end,clip,first,second,deck = SCENES[index]
        progress=(t-start)/(end-start)
        rise=round(24*(1-ease((t-start)/.45)))
        image=self.background.copy()
        self.brand(image)
        if index==0:
            self.text(image,"I'M BUILDING SOMETHING",78,300+rise,25,color=GOLD,bold=True)
            self.text(image,first,72,382+rise,108,bold=True)
            self.text(image,second,72,510+rise,108,color=SPROUT,bold=True)
            self.text(image,deck,78,684+rise,33,color=MUTED)
            source=self.source('collect',1).crop((0,130,1000,830)).resize((932,652),Image.Resampling.LANCZOS).convert('RGBA')
            mask=Image.new('L',source.size,0);ImageDraw.Draw(mask).rounded_rectangle((0,0,931,651),radius=28,fill=255)
            image.paste(source,(74,887+round(12*math.sin(progress*math.pi))),mask)
            ImageDraw.Draw(image).line((80,810,80+round(820*ease(progress*2)),810),fill=GOLD,width=4)
            self.text(image,'AN INFINITE CANVAS FOR YOUR IDEAS',78,1610,24,color=SPROUT,bold=True)
        elif clip:
            self.text(image,first,74,290+rise,84,bold=True)
            self.text(image,second,74,392+rise,84,color=SPROUT,bold=True)
            self.text(image,deck,78,514+rise,29,color=MUTED)
            self.app_window(image,self.source(clip,progress),progress=progress)
            labels={'collect':'01  /  COLLECT YOUR IDEAS','draw':'02  /  MAKE YOUR MARK','share':'03  /  THINK TOGETHER'}
            self.text(image,labels[clip],78,1600,23,color=GOLD,bold=True)
            self.text(image,'iPad + desktop  ·  Self-hosted',78,1650,27,color=MUTED)
        else:
            icon=self.icon.resize((264,220),Image.Resampling.LANCZOS)
            image.alpha_composite(icon,(408,350+rise))
            self.text(image,'BUILT BY ME. SHAPED BY YOU.',0,625+rise,25,color=GOLD,bold=True,centered=True)
            self.text(image,first,0,714+rise,103,bold=True,centered=True)
            self.text(image,second,0,837+rise,112,color=SPROUT,bold=True,centered=True)
            self.text(image,deck,0,1023+rise,36,color=MUTED,centered=True)
            ImageDraw.Draw(image).rounded_rectangle((125,1150,955,1286),radius=36,fill=SPROUT)
            self.text(image,'DM me to test it',0,1183,49,color='#102C1B',bold=True,centered=True)
            self.text(image,'iPad + desktop  ·  Self-hosted',0,1355,29,color=MUTED,centered=True)
            self.text(image,'Your feedback will shape what comes next.',0,1495,27,color=WHITE,centered=True)
        # Keep copy inside the central safe area; only progress lives near the lower edge.
        draw=ImageDraw.Draw(image)
        draw.rounded_rectangle((78,1760,1002,1764),radius=2,fill='#314D39')
        draw.rounded_rectangle((78,1760,78+max(1,round(924*t/DURATION)),1764),radius=2,fill=GOLD)
        self.text(image,'A LITTLE SPACE FOR YOUR THOUGHTS.',78,1790,19,color=MUTED)
        return image.convert('RGB')

    def frame(self,t):
        image=self.scene(t)
        for boundary in [3.2,7.2,11.6,16]:
            if boundary <= t < boundary+.24:
                return Image.blend(self.scene(boundary-.001),image,ease((t-boundary)/.24))
        if t < .2:
            return Image.blend(Image.new('RGB',(WIDTH,HEIGHT),'#081711'),image,ease(t/.2))
        return image


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--capture',type=Path,default=ROOT/'.cache/reel')
    parser.add_argument('--output',type=Path,default=ROOT/'.cache/reel/export')
    parser.add_argument('--font')
    parser.add_argument('--bold-font')
    args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=True)
    reel=Reel(args.capture,args.font,args.bold_font)
    cover=args.output/'mindflow-cover.jpg'
    reel.scene(18.5).save(cover,quality=95)
    # Review all five sections before and after encoding.
    contact=Image.new('RGB',(360*5,640),'#081711')
    for i,t in enumerate([1.6,5.2,9.4,14.3,19]):
        frame=reel.frame(t)
        frame.save(args.output/f'preview-{i+1}.jpg',quality=93)
        contact.paste(frame.resize((360,640),Image.Resampling.LANCZOS),(360*i,0))
    contact.save(args.output/'storyboard.jpg',quality=95)
    output=args.output/'mindflow-announcement.mp4'
    writer=imageio_ffmpeg.write_frames(str(output),(WIDTH,HEIGHT),fps=FPS,codec='libx264',pix_fmt_in='rgb24',pix_fmt_out='yuv420p',quality=None,macro_block_size=1,ffmpeg_log_level='error',output_params=['-crf','19','-preset','medium','-movflags','+faststart'])
    writer.send(None)
    try:
        for n in range(DURATION*FPS):
            writer.send(reel.frame(n/FPS).tobytes())
            if n % (FPS*4)==0: print(f'Rendered {n//FPS}/{DURATION} seconds',flush=True)
    finally:
        writer.close()
    caption="""I've been building mindflow — a private, self-hosted infinite canvas for images, notes, and sketches.\n\nDraw with your mouse or Pencil, collect inspiration, and share a workspace with people you trust.\n\nI'm looking for a few early testers. Interested? Send me a DM and I'll get you set up. I'd love your honest feedback.\n\n#mindflow #buildinpublic #selfhosted\n"""
    (args.output/'caption.txt').write_text(caption)
    (args.output/'README.txt').write_text('mindflow announcement reel\n22 seconds · 1080 × 1920 · 9:16 · 30 fps · H.264 MP4\nSilent with baked-in captions. Optional music can be added in your posting app.\nCover: mindflow-cover.jpg\nSuggested post text: caption.txt\nOnly synthetic local demo data appears in this video.\n')
    with zipfile.ZipFile(args.output/'mindflow-announcement-kit.zip','w',compression=zipfile.ZIP_DEFLATED) as archive:
        for name in ['mindflow-announcement.mp4','mindflow-cover.jpg','caption.txt','README.txt']:
            archive.write(args.output/name,name)
    print(f'Created {output} ({output.stat().st_size/1024/1024:.1f} MiB)',flush=True)


if __name__=='__main__':
    main()
