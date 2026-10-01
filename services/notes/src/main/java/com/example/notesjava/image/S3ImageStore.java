package com.example.notesjava.image;

import io.minio.BucketExistsArgs;
import io.minio.MakeBucketArgs;
import io.minio.MinioClient;
import io.minio.PutObjectArgs;
import io.minio.SetBucketPolicyArgs;

import java.io.ByteArrayInputStream;
import java.util.Locale;

public class S3ImageStore implements ImageStore {

    private static final String READ_POLICY = """
            {
              "Version": "2012-10-17",
              "Statement": [{
                "Effect": "Allow",
                "Principal": {"AWS": ["*"]},
                "Action": ["s3:GetObject"],
                "Resource": ["arn:aws:s3:::%s/*"]
              }]
            }""";

    private final MinioClient client;
    private final String provider;
    private final String bucket;
    private final String publicUrl;
    private volatile boolean bucketReady;

    public S3ImageStore(StorageProperties properties) {
        if (isBlank(properties.endpoint()) || isBlank(properties.accessKey()) || isBlank(properties.secretKey())) {
            throw new IllegalStateException("storage: endpoint, access key and secret key are required");
        }

        this.provider = properties.providerOrDefault();
        if (!StorageProperties.MINIO.equals(provider) && !StorageProperties.R2.equals(provider)) {
            throw new IllegalStateException("storage: unknown provider \"" + provider + "\" (want minio or r2)");
        }
        this.bucket = properties.bucketOrDefault();
        this.publicUrl = publicUrl(properties, provider);

        MinioClient.Builder builder = MinioClient.builder()
                .endpoint(endpointUrl(properties.endpoint(), properties.useSsl()))
                .credentials(properties.accessKey(), properties.secretKey());
        if (StorageProperties.R2.equals(provider)) {
            builder.region("auto");
        }
        this.client = builder.build();
        this.bucketReady = StorageProperties.R2.equals(provider);
    }

    @Override
    public String put(String key, byte[] data, String contentType) {
        try {
            ensureBucket();
            client.putObject(PutObjectArgs.builder()
                    .bucket(bucket)
                    .object(key)
                    .stream(new ByteArrayInputStream(data), data.length, -1)
                    .contentType(contentType)
                    .build());
        } catch (Exception ex) {
            throw new IllegalStateException("storage: put object: " + ex.getMessage(), ex);
        }
        return objectUrl(key);
    }

    String objectUrl(String key) {
        if (StorageProperties.R2.equals(provider)) {
            return publicUrl + "/" + key;
        }
        return publicUrl + "/" + bucket + "/" + key;
    }

    private synchronized void ensureBucket() throws Exception {
        if (bucketReady) {
            return;
        }
        if (!client.bucketExists(BucketExistsArgs.builder().bucket(bucket).build())) {
            client.makeBucket(MakeBucketArgs.builder().bucket(bucket).build());
        }
        client.setBucketPolicy(SetBucketPolicyArgs.builder()
                .bucket(bucket)
                .config(READ_POLICY.formatted(bucket))
                .build());
        bucketReady = true;
    }

    static String publicUrl(StorageProperties properties, String provider) {
        String configured = properties.publicUrl() == null ? "" : properties.publicUrl().trim();
        while (configured.endsWith("/")) {
            configured = configured.substring(0, configured.length() - 1);
        }
        if (!configured.isEmpty()) {
            return configured;
        }
        if (StorageProperties.R2.equals(provider)) {
            throw new IllegalStateException(
                    "storage: a public URL is required for the r2 provider: its S3 endpoint does not serve public reads");
        }
        return endpointUrl(properties.endpoint(), properties.useSsl());
    }

    static String endpointUrl(String endpoint, boolean useSsl) {
        String trimmed = endpoint.trim();
        String lower = trimmed.toLowerCase(Locale.ROOT);
        if (lower.startsWith("http://") || lower.startsWith("https://")) {
            return trimmed;
        }
        return (useSsl ? "https://" : "http://") + trimmed;
    }

    private static boolean isBlank(String value) {
        return value == null || value.isBlank();
    }
}
