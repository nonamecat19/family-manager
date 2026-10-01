package com.example.notesjava.image;

import org.junit.jupiter.api.Test;

import java.nio.charset.StandardCharsets;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class ImageUploadTest {

    static final byte[] PNG = {(byte) 0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n', 0, 0, 0, 13};
    static final byte[] JPEG = {(byte) 0xFF, (byte) 0xD8, (byte) 0xFF, (byte) 0xE0, 0, 16};

    @Test
    void acceptsTheFourImageTypesWhenTheBytesAgree() {
        assertThat(ImageUpload.validate("image/png", PNG)).isEqualTo(new ImageUpload.Accepted("image/png", ".png"));
        assertThat(ImageUpload.validate(" IMAGE/JPEG ", JPEG).extension()).isEqualTo(".jpg");
        assertThat(ImageUpload.validate("image/gif", "GIF89a...".getBytes(StandardCharsets.US_ASCII)).extension())
                .isEqualTo(".gif");
        assertThat(ImageUpload.validate("image/webp", "RIFF\0\0\0\0WEBPVP8 ".getBytes(StandardCharsets.US_ASCII))
                .extension()).isEqualTo(".webp");
    }

    @Test
    void refusesAnUnsupportedContentType() {
        assertThatThrownBy(() -> ImageUpload.validate("image/svg+xml", PNG))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("unsupported content_type");
        assertThatThrownBy(() -> ImageUpload.validate("", PNG)).isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void refusesBytesThatDoNotMatchTheDeclaredType() {
        assertThatThrownBy(() -> ImageUpload.validate("image/png", JPEG))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("not a image/png");
        assertThatThrownBy(() -> ImageUpload.validate("image/jpeg", "<html>".getBytes(StandardCharsets.US_ASCII)))
                .isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void refusesAnEmptyOrOversizedImage() {
        assertThatThrownBy(() -> ImageUpload.validate("image/png", new byte[0]))
                .hasMessageContaining("image is required");

        byte[] huge = new byte[ImageUpload.MAX_BYTES + 1];
        System.arraycopy(PNG, 0, huge, 0, PNG.length);
        assertThatThrownBy(() -> ImageUpload.validate("image/png", huge)).hasMessageContaining("8MB");
    }

    @Test
    void theKeyIsContentAddressedUnderTheFamilyAndNote() {
        UUID family = UUID.fromString("22222222-2222-2222-2222-222222222222");

        String key = ImageUpload.key(family, 42L, PNG, ".png");

        assertThat(key).matches("22222222-2222-2222-2222-222222222222/42/[0-9a-f]{32}\\.png");
        assertThat(ImageUpload.key(family, 42L, PNG, ".png")).isEqualTo(key);
        assertThat(ImageUpload.key(family, 42L, JPEG, ".png")).isNotEqualTo(key);
    }
}
