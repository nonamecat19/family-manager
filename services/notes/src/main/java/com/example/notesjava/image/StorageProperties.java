package com.example.notesjava.image;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.util.Locale;

@ConfigurationProperties(prefix = "notes.storage")
public record StorageProperties(
        String provider,
        String endpoint,
        String accessKey,
        String secretKey,
        boolean useSsl,
        String bucket,
        String publicUrl) {

    public static final String MINIO = "minio";
    public static final String R2 = "r2";

    public String providerOrDefault() {
        return provider == null || provider.isBlank() ? MINIO : provider.trim().toLowerCase(Locale.ROOT);
    }

    public String bucketOrDefault() {
        return bucket == null || bucket.isBlank() ? "notes" : bucket.trim();
    }
}
