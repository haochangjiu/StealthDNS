#!/bin/bash

# StealthDNS App Icon Generator
# Usage: ./generate_icons.sh <path_to_1024x1024_icon.png>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [ -z "$1" ]; then
    echo "Usage: $0 <icon_file_path>"
    echo "Example: $0 ~/Downloads/icon.png"
    echo ""
    echo "Please provide a 1024x1024 PNG icon file"
    exit 1
fi

SOURCE_ICON="$1"

if [ ! -f "$SOURCE_ICON" ]; then
    echo "Error: File not found: $SOURCE_ICON"
    exit 1
fi

# Check if ImageMagick is installed
if ! command -v convert &> /dev/null; then
    echo "ImageMagick is required"
    echo "macOS: brew install imagemagick"
    echo "Ubuntu: sudo apt install imagemagick"
    exit 1
fi

echo "Generating app icons..."

# Get image dimensions
WIDTH=$(identify -format "%w" "$SOURCE_ICON")
HEIGHT=$(identify -format "%h" "$SOURCE_ICON")

echo "Source image size: ${WIDTH}x${HEIGHT}"

# If not square, create a square version by padding
if [ "$WIDTH" != "$HEIGHT" ]; then
    echo "Image is not square, creating square version..."
    # Determine the larger dimension
    if [ "$WIDTH" -gt "$HEIGHT" ]; then
        SIZE=$WIDTH
    else
        SIZE=$HEIGHT
    fi
    # Create a temporary square image with transparent background, centered
    TEMP_ICON="/tmp/appicon_square_temp.png"
    convert "$SOURCE_ICON" -gravity center -background transparent -extent ${SIZE}x${SIZE} "$TEMP_ICON"
    SOURCE_ICON="$TEMP_ICON"
    echo "Created square image: ${SIZE}x${SIZE}"
fi

# Android Icons
ANDROID_RES="$PROJECT_DIR/android/app/src/main/res"

echo "Generating Android icons..."
mkdir -p "$ANDROID_RES/mipmap-hdpi"
mkdir -p "$ANDROID_RES/mipmap-xhdpi"
mkdir -p "$ANDROID_RES/mipmap-xxhdpi"
mkdir -p "$ANDROID_RES/mipmap-xxxhdpi"

# Delete old XML icon files to avoid duplicate resource errors
rm -f "$ANDROID_RES/mipmap-hdpi/ic_launcher.xml" "$ANDROID_RES/mipmap-hdpi/ic_launcher_round.xml"
rm -f "$ANDROID_RES/mipmap-xhdpi/ic_launcher.xml" "$ANDROID_RES/mipmap-xhdpi/ic_launcher_round.xml"
rm -f "$ANDROID_RES/mipmap-xxhdpi/ic_launcher.xml" "$ANDROID_RES/mipmap-xxhdpi/ic_launcher_round.xml"
rm -f "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher.xml" "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher_round.xml"
# Delete adaptive icon configuration to ensure PNG icons are used
rm -rf "$ANDROID_RES/mipmap-anydpi-v26"

convert "$SOURCE_ICON" -resize 72x72 "$ANDROID_RES/mipmap-hdpi/ic_launcher.png"
convert "$SOURCE_ICON" -resize 72x72 "$ANDROID_RES/mipmap-hdpi/ic_launcher_round.png"
convert "$SOURCE_ICON" -resize 96x96 "$ANDROID_RES/mipmap-xhdpi/ic_launcher.png"
convert "$SOURCE_ICON" -resize 96x96 "$ANDROID_RES/mipmap-xhdpi/ic_launcher_round.png"
convert "$SOURCE_ICON" -resize 144x144 "$ANDROID_RES/mipmap-xxhdpi/ic_launcher.png"
convert "$SOURCE_ICON" -resize 144x144 "$ANDROID_RES/mipmap-xxhdpi/ic_launcher_round.png"
convert "$SOURCE_ICON" -resize 192x192 "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher.png"
convert "$SOURCE_ICON" -resize 192x192 "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher_round.png"

echo "Android icons generated ✓"

# iOS Icons
IOS_ASSETS="$PROJECT_DIR/ios/StealthDNS/Assets.xcassets/AppIcon.appiconset"

echo "Generating iOS icons..."
mkdir -p "$IOS_ASSETS"

# iPhone icons
convert "$SOURCE_ICON" -resize 40x40 "$IOS_ASSETS/Icon-40.png"
convert "$SOURCE_ICON" -resize 60x60 "$IOS_ASSETS/Icon-60.png"
convert "$SOURCE_ICON" -resize 58x58 "$IOS_ASSETS/Icon-58.png"
convert "$SOURCE_ICON" -resize 87x87 "$IOS_ASSETS/Icon-87.png"
convert "$SOURCE_ICON" -resize 80x80 "$IOS_ASSETS/Icon-80.png"
convert "$SOURCE_ICON" -resize 120x120 "$IOS_ASSETS/Icon-120.png"
convert "$SOURCE_ICON" -resize 180x180 "$IOS_ASSETS/Icon-180.png"

# iPad icons
convert "$SOURCE_ICON" -resize 20x20 "$IOS_ASSETS/Icon-20.png"
convert "$SOURCE_ICON" -resize 29x29 "$IOS_ASSETS/Icon-29.png"
convert "$SOURCE_ICON" -resize 76x76 "$IOS_ASSETS/Icon-76.png"
convert "$SOURCE_ICON" -resize 152x152 "$IOS_ASSETS/Icon-152.png"
convert "$SOURCE_ICON" -resize 167x167 "$IOS_ASSETS/Icon-167.png"

# App Store icon
convert "$SOURCE_ICON" -resize 1024x1024 "$IOS_ASSETS/Icon-1024.png"

# Update Contents.json
cat > "$IOS_ASSETS/Contents.json" << 'EOF'
{
  "images" : [
    {
      "filename" : "Icon-40.png",
      "idiom" : "iphone",
      "scale" : "2x",
      "size" : "20x20"
    },
    {
      "filename" : "Icon-60.png",
      "idiom" : "iphone",
      "scale" : "3x",
      "size" : "20x20"
    },
    {
      "filename" : "Icon-58.png",
      "idiom" : "iphone",
      "scale" : "2x",
      "size" : "29x29"
    },
    {
      "filename" : "Icon-87.png",
      "idiom" : "iphone",
      "scale" : "3x",
      "size" : "29x29"
    },
    {
      "filename" : "Icon-80.png",
      "idiom" : "iphone",
      "scale" : "2x",
      "size" : "40x40"
    },
    {
      "filename" : "Icon-120.png",
      "idiom" : "iphone",
      "scale" : "3x",
      "size" : "40x40"
    },
    {
      "filename" : "Icon-120.png",
      "idiom" : "iphone",
      "scale" : "2x",
      "size" : "60x60"
    },
    {
      "filename" : "Icon-180.png",
      "idiom" : "iphone",
      "scale" : "3x",
      "size" : "60x60"
    },
    {
      "filename" : "Icon-20.png",
      "idiom" : "ipad",
      "scale" : "1x",
      "size" : "20x20"
    },
    {
      "filename" : "Icon-40.png",
      "idiom" : "ipad",
      "scale" : "2x",
      "size" : "20x20"
    },
    {
      "filename" : "Icon-29.png",
      "idiom" : "ipad",
      "scale" : "1x",
      "size" : "29x29"
    },
    {
      "filename" : "Icon-58.png",
      "idiom" : "ipad",
      "scale" : "2x",
      "size" : "29x29"
    },
    {
      "filename" : "Icon-40.png",
      "idiom" : "ipad",
      "scale" : "1x",
      "size" : "40x40"
    },
    {
      "filename" : "Icon-80.png",
      "idiom" : "ipad",
      "scale" : "2x",
      "size" : "40x40"
    },
    {
      "filename" : "Icon-76.png",
      "idiom" : "ipad",
      "scale" : "1x",
      "size" : "76x76"
    },
    {
      "filename" : "Icon-152.png",
      "idiom" : "ipad",
      "scale" : "2x",
      "size" : "76x76"
    },
    {
      "filename" : "Icon-167.png",
      "idiom" : "ipad",
      "scale" : "2x",
      "size" : "83.5x83.5"
    },
    {
      "filename" : "Icon-1024.png",
      "idiom" : "ios-marketing",
      "scale" : "1x",
      "size" : "1024x1024"
    }
  ],
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
EOF

echo "iOS icons generated ✓"

echo ""
echo "========================================="
echo "Icon generation complete!"
echo "========================================="
echo ""
echo "Android icons location: $ANDROID_RES/mipmap-*/"
echo "iOS icons location: $IOS_ASSETS/"
echo ""
echo "Run the following commands to rebuild the app:"
echo "  make android  # Build Android"
echo "  make ios      # Build iOS"
