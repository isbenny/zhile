#!/usr/bin/env python3
"""Write the self-hosted APK feed after building and verifying a release."""
import argparse
import hashlib
import json
from pathlib import Path

ORIGIN = 'https://music.cnxwkj.com'
NAMES = ('zhile-arm64.apk', 'zhile-x86_64.apk', 'zhile-universal.apk')

def create_manifest(directory, version):
    import re
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?', version):
        raise ValueError('Version must be a numeric release version')
    assets = []
    for name in NAMES:
        path = directory / name
        if not path.is_file() or not path.stat().st_size:
            raise ValueError(f'Missing release APK: {name}')
        with path.open('rb') as stream:
            sha = hashlib.file_digest(stream, 'sha256').hexdigest()
        assets.append({'name': name, 'size': path.stat().st_size,
                       'content_type': 'application/vnd.android.package-archive',
                       'browser_download_url': f'{ORIGIN}/downloads/{name}', 'sha256': sha})
    if not (directory / 'zhile-source.zip').is_file():
        raise ValueError('Corresponding source must accompany the APKs')
    return {'tag_name': 'v' + version, 'name': '知乐 v' + version,
            'html_url': ORIGIN + '/', 'body': '搜索、播放、歌单与移动端体验更新。请查看下载页的版本说明。',
            'source_url': ORIGIN + '/downloads/zhile-source.zip', 'assets': assets}

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    parser.add_argument('--version', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    payload = create_manifest(args.directory, args.version)
    args.output.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n')
