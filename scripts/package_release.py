#!/usr/bin/env python3
"""Package tracked source + dependency sources and a self-hosted Zhile download site.
Run with Python 3 and Go 1.27 from a clean checkout after placing APKs in --output.
"""
import argparse, hashlib, json, os, shutil, subprocess, tempfile, urllib.request, zipfile, re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION = re.search(r'AppVersion = "([^"]+)"', (ROOT / 'core/version.go').read_text()).group(1)
FFMPEG_URL = 'https://ffmpeg.org/releases/ffmpeg-8.0.1.tar.xz'
FFMPEG_SHA = '05ee0b03119b45c0bdb4df654b96802e909e0a752f72e4fe3794f487229e5a41'

def run(args, cwd=ROOT):
    return subprocess.check_output(args, cwd=cwd, text=True)

def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()

def escaped(value):
    return ''.join('!' + c.lower() if c.isupper() else c for c in value)

def module_records(cwd):
    raw = run(['go', 'mod', 'download', '-json', 'all'], cwd)
    decoder = json.JSONDecoder()
    while raw.strip():
        item, end = decoder.raw_decode(raw.lstrip())
        raw = raw.lstrip()[end:]
        if item.get('Error'):
            raise RuntimeError(item['Error'])
        yield item

def source_bundle(output):
    target = output / 'zhile-source.zip'
    revision = run(['git', 'rev-parse', 'HEAD']).strip()
    run(['git', 'archive', '--format=zip', '--prefix=zhile/', '-o', str(target), revision])
    records, seen = [], set()
    with zipfile.ZipFile(target, 'a', compression=zipfile.ZIP_DEFLATED) as z:
        for folder in (ROOT, ROOT / 'tools/gio-cmd'):
            for item in module_records(folder):
                key = (item['Path'], item['Version'])
                if key in seen:
                    continue
                seen.add(key)
                prefix = f"dependency-sources/{escaped(item['Path'])}/@v/{escaped(item['Version'])}"
                files = {}
                for field, suffix in (('Zip', '.zip'), ('GoMod', '.mod'), ('Info', '.info')):
                    src = Path(item[field])
                    relative = prefix + suffix
                    z.write(src, 'zhile/' + relative)
                    files[field] = relative
                records.append({'module': key[0], 'version': key[1], 'sum': item.get('Sum'), 'go_mod_sum': item.get('GoModSum'), 'files': files})
        with tempfile.TemporaryDirectory() as td:
            archive = Path(td) / 'ffmpeg-8.0.1.tar.xz'
            cached = Path('/tmp/ffmpeg-8.0.1.tar.xz')
            if cached.is_file() and digest(cached) == FFMPEG_SHA:
                shutil.copyfile(cached, archive)
            else:
                urllib.request.urlretrieve(FFMPEG_URL, archive)
            if digest(archive) != FFMPEG_SHA:
                raise RuntimeError('FFmpeg source checksum mismatch')
            z.write(archive, 'zhile/dependency-sources/ffmpeg-8.0.1.tar.xz')
        z.writestr('zhile/dependency-sources/manifest.json', json.dumps(records, ensure_ascii=False, indent=2))
        z.writestr('zhile/SOURCE-REVISION.txt', f'Zhile v{VERSION}\nRepository: https://github.com/isbenny/zhile\nCommit: {revision}\nFFmpeg source: {FFMPEG_URL}\nSHA256: {FFMPEG_SHA}\n')
    print(f'Source bundle: {target.name}; {len(records)} pinned dependency source archives.', flush=True)

def site_bundle(output):
    assets = ['zhile-arm64.apk', 'zhile-universal.apk', 'zhile-x86_64.apk', 'zhile-source.zip']
    for name in assets:
        if not (output / name).is_file():
            raise FileNotFoundError(output / name)
    checksum = output / 'SHA256SUMS.txt'
    checksum.write_text(''.join(f'{digest(output / name)}  {name}\n' for name in assets))
    site = output / 'website'
    if site.exists():
        raise FileExistsError(f'Remove or relocate the previous generated site before rerunning: {site}')
    shutil.copytree(ROOT / 'website/dist', site)
    (site / 'downloads').mkdir()
    for name in assets + ['SHA256SUMS.txt']:
        shutil.copyfile(output / name, site / 'downloads' / name)
    index = site / 'index.html'
    index.write_text(index.read_text().replace(f'https://github.com/isbenny/zhile/releases/download/v{VERSION}/', 'downloads/'))
    instructions = f'''知乐 Zhile v{VERSION} — 服务器上传说明

把 website 文件夹里的所有内容整体上传到网站目录，保留 assets 和 downloads 的相对位置。
例如：网站根目录/index.html、assets/、downloads/、LICENSE.txt、NOTICE.txt。
手机打开你的网站地址，点“下载 Android 版”，按系统提示安装即可。
本网页完全静态，不需要配置 Go、数据库或登录。
请保留源码下载、许可证和版权说明。不要把签名密钥上传到网站或 GitHub。
APK 与 Go/FFmpeg 构建、签名及浏览器布局检查已完成，尚未在安卓真机上验证。
大多数手机用 arm64 版；不兼容时用通用版。最低 Android 7.0。
'''
    (output / '使用与上传说明.txt').write_text(instructions)
    target = output / 'zhile-website.zip'
    with zipfile.ZipFile(target, 'w', zipfile.ZIP_DEFLATED) as z:
        for p in sorted(site.rglob('*')):
            if p.is_file():
                z.write(p, p.relative_to(site).as_posix())
    print(f'Self-hosted download site: {target}', flush=True)

parser = argparse.ArgumentParser()
parser.add_argument('--output', type=Path, required=True)
parser.add_argument('--stage', choices=['source', 'site', 'all'], default='all')
args = parser.parse_args()
args.output = args.output.expanduser().resolve()
args.output.mkdir(parents=True, exist_ok=True)
if args.stage in ('source', 'all'):
    source_bundle(args.output)
if args.stage in ('site', 'all'):
    site_bundle(args.output)
