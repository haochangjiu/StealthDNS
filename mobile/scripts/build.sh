#!/bin/bash

# StealthDNS Mobile Browser Build Script
# Builds the mobile browser app for Android and iOS using native development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
GOLIB_DIR="$PROJECT_DIR/golib"
OUTPUT_DIR="$PROJECT_DIR/build"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --golib          Build Go NHP core library (requires gomobile)"
    echo "  --android        Build Android APK"
    echo "  --ios            Build iOS app"
    echo "  --all            Build everything"
    echo "  --release        Build release version (default)"
    echo "  --debug          Build debug version"
    echo "  --clean          Clean build artifacts"
    echo "  --help           Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --golib --android    # Build Go lib and Android APK"
    echo "  $0 --all --release      # Build everything in release mode"
}

check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_info "Go: $(go version | cut -d' ' -f3)"
}

check_gomobile() {
    if ! command -v gomobile &> /dev/null; then
        print_info "Installing gomobile..."
        go install golang.org/x/mobile/cmd/gomobile@latest
        go install golang.org/x/mobile/cmd/gobind@latest
    fi
    
    # Initialize gomobile
    gomobile init 2>/dev/null || true
    print_info "gomobile is ready"
}

check_android_sdk() {
    if [ -z "$ANDROID_HOME" ] && [ -z "$ANDROID_SDK_ROOT" ]; then
        print_warning "ANDROID_HOME not set, Android builds may fail"
        return 1
    fi
    print_info "Android SDK: ${ANDROID_HOME:-$ANDROID_SDK_ROOT}"
    return 0
}

check_xcode() {
    if ! command -v xcodebuild &> /dev/null; then
        print_warning "Xcode not installed, iOS builds not available"
        return 1
    fi
    print_info "Xcode: $(xcodebuild -version | head -1)"
    return 0
}

clean_build() {
    print_info "Cleaning build artifacts..."
    rm -rf "$OUTPUT_DIR"
    rm -rf "$PROJECT_DIR/android/app/build"
    rm -rf "$PROJECT_DIR/android/app/libs/*.aar"
    rm -rf "$PROJECT_DIR/ios/build"
    print_success "Clean complete"
}

build_golib_android() {
    print_info "Building Go library for Android..."
    
    cd "$GOLIB_DIR"
    
    mkdir -p "$OUTPUT_DIR/golib"
    mkdir -p "$PROJECT_DIR/android/app/libs"
    
    gomobile bind \
        -target=android \
        -androidapi=21 \
        -o "$OUTPUT_DIR/golib/nhpcore.aar" \
        .
    
    # Copy to Android libs folder
    cp "$OUTPUT_DIR/golib/nhpcore.aar" "$PROJECT_DIR/android/app/libs/"
    
    print_success "Android Go library: $OUTPUT_DIR/golib/nhpcore.aar"
}

build_golib_ios() {
    print_info "Building Go library for iOS..."
    
    cd "$GOLIB_DIR"
    
    mkdir -p "$OUTPUT_DIR/golib"
    
    gomobile bind \
        -target=ios \
        -o "$OUTPUT_DIR/golib/Nhpcore.xcframework" \
        .
    
    # Copy to iOS project
    cp -r "$OUTPUT_DIR/golib/Nhpcore.xcframework" "$PROJECT_DIR/ios/"
    
    print_success "iOS Go library: $OUTPUT_DIR/golib/Nhpcore.xcframework"
}

build_android() {
    local build_type="${1:-release}"
    
    print_info "Building Android APK ($build_type)..."
    
    cd "$PROJECT_DIR/android"
    
    # Check if nhpcore.aar exists
    if [ ! -f "app/libs/nhpcore.aar" ]; then
        print_warning "nhpcore.aar not found, building Go library first..."
        build_golib_android
    fi
    
    mkdir -p "$OUTPUT_DIR/android"
    
    if [ "$build_type" = "release" ]; then
        ./gradlew assembleRelease
        cp app/build/outputs/apk/release/app-release.apk "$OUTPUT_DIR/android/StealthDNS-Browser.apk" 2>/dev/null || \
        cp app/build/outputs/apk/release/app-release-unsigned.apk "$OUTPUT_DIR/android/StealthDNS-Browser.apk"
    else
        ./gradlew assembleDebug
        cp app/build/outputs/apk/debug/app-debug.apk "$OUTPUT_DIR/android/StealthDNS-Browser-debug.apk"
    fi
    
    print_success "Android APK: $OUTPUT_DIR/android/"
}

build_android_bundle() {
    print_info "Building Android App Bundle..."
    
    cd "$PROJECT_DIR/android"
    
    mkdir -p "$OUTPUT_DIR/android"
    
    ./gradlew bundleRelease
    cp app/build/outputs/bundle/release/app-release.aab "$OUTPUT_DIR/android/StealthDNS-Browser.aab"
    
    print_success "Android AAB: $OUTPUT_DIR/android/StealthDNS-Browser.aab"
}

build_ios() {
    local build_type="${1:-release}"
    
    print_info "Building iOS app ($build_type)..."
    
    cd "$PROJECT_DIR/ios"
    
    # Check if Nhpcore.xcframework exists
    if [ ! -d "Nhpcore.xcframework" ]; then
        print_warning "Nhpcore.xcframework not found, building Go library first..."
        build_golib_ios
    fi
    
    mkdir -p "$OUTPUT_DIR/ios"
    
    # Build configuration
    local config="Release"
    if [ "$build_type" = "debug" ]; then
        config="Debug"
    fi
    
    xcodebuild \
        -project StealthDNS.xcodeproj \
        -scheme StealthDNS \
        -configuration "$config" \
        -destination 'generic/platform=iOS' \
        -archivePath "$OUTPUT_DIR/ios/StealthDNS.xcarchive" \
        clean archive \
        CODE_SIGN_IDENTITY="" \
        CODE_SIGNING_REQUIRED=NO \
        CODE_SIGNING_ALLOWED=NO
    
    print_success "iOS archive: $OUTPUT_DIR/ios/StealthDNS.xcarchive"
    print_info "Use Xcode to export IPA with proper signing"
}

# Parse arguments
BUILD_GOLIB=false
BUILD_ANDROID=false
BUILD_IOS=false
BUILD_TYPE="release"
DO_CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --golib) BUILD_GOLIB=true; shift ;;
        --android) BUILD_ANDROID=true; shift ;;
        --ios) BUILD_IOS=true; shift ;;
        --all) BUILD_GOLIB=true; BUILD_ANDROID=true; BUILD_IOS=true; shift ;;
        --release) BUILD_TYPE="release"; shift ;;
        --debug) BUILD_TYPE="debug"; shift ;;
        --clean) DO_CLEAN=true; shift ;;
        --help) show_usage; exit 0 ;;
        *) print_error "Unknown option: $1"; show_usage; exit 1 ;;
    esac
done

# Main
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       StealthDNS Mobile Browser Build Script              ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

if [ "$DO_CLEAN" = true ]; then
    clean_build
fi

if [ "$BUILD_GOLIB" = false ] && [ "$BUILD_ANDROID" = false ] && [ "$BUILD_IOS" = false ]; then
    print_error "No build target specified"
    show_usage
    exit 1
fi

# Build Go library
if [ "$BUILD_GOLIB" = true ]; then
    check_go
    check_gomobile
    
    if [ "$BUILD_ANDROID" = true ] || check_android_sdk; then
        build_golib_android
    fi
    
    if [ "$BUILD_IOS" = true ] && check_xcode; then
        build_golib_ios
    fi
fi

# Build Android
if [ "$BUILD_ANDROID" = true ]; then
    if check_android_sdk; then
        build_android "$BUILD_TYPE"
    fi
fi

# Build iOS
if [ "$BUILD_IOS" = true ]; then
    if check_xcode; then
        build_ios "$BUILD_TYPE"
    fi
fi

# Summary
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                     Build Summary                         ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
ls -la "$OUTPUT_DIR"/**/* 2>/dev/null || echo "  No outputs"
echo ""
print_success "Build complete!"
