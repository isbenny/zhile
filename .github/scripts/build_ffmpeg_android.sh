#!/usr/bin/env bash
# Melofern: build redistributable audio tools from matching source (2026-10-30).
set -euo pipefail
abi="${1:?Android ABI required}"
case "$abi" in
  arm64-v8a) arch=aarch64; triple=aarch64-linux-android; compiler=aarch64-linux-android24 ;;
  armeabi-v7a) arch=arm; triple=arm-linux-androideabi; compiler=armv7a-linux-androideabi24 ;;
  x86) arch=x86; triple=i686-linux-android; compiler=i686-linux-android24 ;;
  x86_64) arch=x86_64; triple=x86_64-linux-android; compiler=x86_64-linux-android24 ;;
  *) exit 2 ;;
esac
root="$PWD"
toolchain="$ANDROID_NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64"
export PATH="$toolchain/bin:$PATH"
mkdir -p build/ffmpeg "$root/desktop_app/ffmpeg/android/$abi"
curl --fail --location --retry 3 https://ffmpeg.org/releases/ffmpeg-8.0.1.tar.xz -o build/ffmpeg/source.tar.xz
printf '%s  %s\n' 05ee0b03119b45c0bdb4df654b96802e909e0a752f72e4fe3794f487229e5a41 build/ffmpeg/source.tar.xz | sha256sum --check
cd build/ffmpeg
tar -xf source.tar.xz
cd ffmpeg-8.0.1
./configure --target-os=android --arch="$arch" --enable-cross-compile \
  --cc="$compiler-clang" --cxx="$compiler-clang++" \
  --ar=llvm-ar --ranlib=llvm-ranlib --strip=llvm-strip --nm=llvm-nm \
  --sysroot="$toolchain/sysroot" --enable-pic --enable-static --disable-shared \
  --disable-autodetect --disable-asm --disable-doc --disable-debug --disable-ffplay \
  --disable-network --extra-ldflags='-Wl,-z,max-page-size=16384' \
  --prefix="$root/build/ffmpeg/install-$abi"
make -j"$(nproc)"
llvm-strip ffmpeg ffprobe
cp ffmpeg ffprobe "$root/desktop_app/ffmpeg/android/$abi/"
# No external codec libraries or C++ runtime are linked into these executables.
llvm-readelf -d ffmpeg | grep NEEDED
cp COPYING.LGPLv2.1 "$root/desktop_app/ffmpeg/android/$abi/FFMPEG-LICENSE.txt"
