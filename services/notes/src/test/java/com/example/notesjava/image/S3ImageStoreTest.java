package com.example.notesjava.image;

import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class S3ImageStoreTest {

    private static StorageProperties props(String provider, String publicUrl, boolean ssl) {
        return new StorageProperties(provider, "minio:9000", "key", "secret", ssl, "notes", publicUrl);
    }

    @Test
    void minioObjectsAreServedUnderTheBucket() {
        S3ImageStore store = new S3ImageStore(props("minio", "http://localhost:9000/", false));

        assertThat(store.objectUrl("fam/1/abc.png")).isEqualTo("http://localhost:9000/notes/fam/1/abc.png");
    }

    @Test
    void minioFallsBackToTheEndpointForItsPublicUrl() {
        assertThat(new S3ImageStore(props("", "", false)).objectUrl("k.png"))
                .isEqualTo("http://minio:9000/notes/k.png");
        assertThat(new S3ImageStore(props("minio", null, true)).objectUrl("k.png"))
                .isEqualTo("https://minio:9000/notes/k.png");
    }

    @Test
    void r2ObjectsAreServedFromTheBucketDomainAndNeedAPublicUrl() {
        assertThat(new S3ImageStore(props("r2", "https://images.example.test", true)).objectUrl("k.png"))
                .isEqualTo("https://images.example.test/k.png");
        assertThatThrownBy(() -> new S3ImageStore(props("r2", "", true)))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("public URL");
    }

    @Test
    void refusesAnUnknownProviderOrMissingCredentials() {
        assertThatThrownBy(() -> new S3ImageStore(props("s3", "", false)))
                .isInstanceOf(IllegalStateException.class);
        assertThatThrownBy(() -> new S3ImageStore(
                new StorageProperties("minio", "minio:9000", "", "secret", false, "notes", "")))
                .isInstanceOf(IllegalStateException.class);
    }
}
