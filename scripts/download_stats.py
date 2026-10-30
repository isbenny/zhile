#!/usr/bin/env python3
"""Summarize APK HTTP responses; these are not unique downloads or installs."""
import argparse
import collections
import json
import re
from pathlib import Path

APK = re.compile(r'^/downloads/zhile-(arm64|x86_64|universal)\.apk$')

def summarize(lines):
    counts = collections.defaultdict(lambda: {'responses': 0, 'bytes': 0, 'errors': 0})
    invalid = 0
    for line in lines:
        try:
            item = json.loads(line)
            if not APK.fullmatch(item['path']):
                continue
            day = item['time'][:10]
            status, size = int(item['status']), int(item['bytes'])
            if not re.fullmatch(r'\d{4}-\d{2}-\d{2}', day) or size < 0:
                raise ValueError('Invalid log row')
        except (ValueError, TypeError, KeyError):
            invalid += 1
            continue
        key = (day, item['path'].rsplit('/', 1)[-1])
        if status in (200, 206):
            counts[key]['responses'] += 1
            counts[key]['bytes'] += size
        elif status >= 400:
            counts[key]['errors'] += 1
    return counts, invalid

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('log', type=Path)
    args = parser.parse_args()
    with args.log.open() as stream:
        rows, invalid = summarize(stream)
    print('日期\t安装包\t成功响应数(非安装量)\t传输MiB\t错误数')
    for (day, name), row in sorted(rows.items()):
        print(f"{day}\t{name}\t{row['responses']}\t{row['bytes']/1048576:.2f}\t{row['errors']}")
    if invalid:
        print(f'忽略无效日志行：{invalid}')
