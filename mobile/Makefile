# StealthDNS Mobile Browser Makefile
# Native Android (Kotlin) and iOS (Swift) with Go NHP Core

.PHONY: all clean golib golib-android golib-ios android android-debug ios ios-debug help

# Default
all: golib android ios

help:
	@echo "StealthDNS Mobile Browser Build System"
	@echo ""
	@echo "Targets:"
	@echo "  golib           Build Go NHP core for all platforms"
	@echo "  golib-android   Build Go NHP core for Android (.aar)"
	@echo "  golib-ios       Build Go NHP core for iOS (.xcframework)"
	@echo "  android         Build Android APK (release)"
	@echo "  android-debug   Build Android APK (debug)"
	@echo "  android-bundle  Build Android App Bundle"
	@echo "  ios             Build iOS (release)"
	@echo "  ios-debug       Build iOS (debug)"
	@echo "  clean           Clean build artifacts"
	@echo "  all             Build everything"
	@echo ""

# Clean
clean:
	@echo "Cleaning..."
	rm -rf build
	rm -rf android/app/build
	rm -rf android/app/libs/*.aar
	rm -rf ios/build
	rm -rf ios/Nhpcore.xcframework

# Go NHP Core Library
golib: golib-android golib-ios

golib-android:
	@echo "Building Go library for Android..."
	@mkdir -p build/golib android/app/libs
	cd golib && gomobile bind -target=android -androidapi=21 -o ../build/golib/nhpcore.aar .
	cp build/golib/nhpcore.aar android/app/libs/

golib-ios:
	@echo "Building Go library for iOS..."
	@mkdir -p build/golib
	cd golib && gomobile bind -target=ios -o ../build/golib/Nhpcore.xcframework .
	cp -r build/golib/Nhpcore.xcframework ios/

# Android
android: golib-android
	@echo "Building Android APK (release)..."
	@if [ ! -f android/gradlew ]; then echo "Error: Run 'make' from the mobile directory"; exit 1; fi
	@mkdir -p build/android
	cd android && chmod +x gradlew && ./gradlew assembleRelease
	cp android/app/build/outputs/apk/release/*.apk build/android/StealthDNS-Browser.apk 2>/dev/null || true
	@echo "Output: build/android/StealthDNS-Browser.apk"

android-debug:
	@echo "Building Android APK (debug)..."
	@if [ ! -f android/gradlew ]; then echo "Error: Run 'make' from the mobile directory"; exit 1; fi
	@mkdir -p build/android
	cd android && chmod +x gradlew && ./gradlew assembleDebug
	cp android/app/build/outputs/apk/debug/*.apk build/android/StealthDNS-Browser-debug.apk
	@echo "Output: build/android/StealthDNS-Browser-debug.apk"

android-bundle: golib-android
	@echo "Building Android App Bundle..."
	@if [ ! -f android/gradlew ]; then echo "Error: Run 'make' from the mobile directory"; exit 1; fi
	@mkdir -p build/android
	cd android && chmod +x gradlew && ./gradlew bundleRelease
	cp android/app/build/outputs/bundle/release/*.aab build/android/StealthDNS-Browser.aab
	@echo "Output: build/android/StealthDNS-Browser.aab"

# iOS
ios: golib-ios
	@echo "Building iOS (release)..."
	@mkdir -p build/ios
	cd ios && xcodebuild -project StealthDNS.xcodeproj -scheme StealthDNS -configuration Release \
		-destination 'generic/platform=iOS' -archivePath ../build/ios/StealthDNS.xcarchive \
		archive CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED=NO
	@echo "Output: build/ios/StealthDNS.xcarchive"
	@echo "Use Xcode to export IPA with signing"

ios-debug:
	@echo "Building iOS (debug)..."
	cd ios && xcodebuild -project StealthDNS.xcodeproj -scheme StealthDNS -configuration Debug \
		-destination 'platform=iOS Simulator,name=iPhone 15' build

# Install gomobile
setup-gomobile:
	go install golang.org/x/mobile/cmd/gomobile@latest
	go install golang.org/x/mobile/cmd/gobind@latest
	gomobile init
