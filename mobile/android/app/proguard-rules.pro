# ProGuard rules for StealthDNS Browser

# Keep NHP Core library classes
-keep class nhpcore.** { *; }
-keep class go.** { *; }

# Keep Kotlin metadata
-keep class kotlin.Metadata { *; }
-keepclassmembers class kotlin.Metadata {
    public <methods>;
}

# WebView
-keepclassmembers class * extends android.webkit.WebViewClient {
    public void *(android.webkit.WebView, java.lang.String, android.graphics.Bitmap);
    public boolean *(android.webkit.WebView, java.lang.String);
}

# Keep JSON classes
-keepclassmembers class * {
    @org.json.* <fields>;
}

# AndroidX
-keep class androidx.** { *; }
-keep interface androidx.** { *; }
