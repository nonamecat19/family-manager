package com.example.notesjava.image;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;

public final class ImageUpload {

    public static final int MAX_BYTES = 8 << 20;

    private static final Map<String, String> EXTENSIONS = Map.of(
            "image/jpeg", ".jpg",
            "image/png", ".png",
            "image/webp", ".webp",
            "image/gif", ".gif");

    private ImageUpload() {
    }

    public record Accepted(String contentType, String extension) {
    }

    public static Accepted validate(String contentType, byte[] data) {
        String type = contentType == null ? "" : contentType.trim().toLowerCase(Locale.ROOT);
        String extension = EXTENSIONS.get(type);
        if (extension == null) {
            throw new IllegalArgumentException("unsupported content_type \"" + type + "\"");
        }
        if (data == null || data.length == 0) {
            throw new IllegalArgumentException("image is required");
        }
        if (data.length > MAX_BYTES) {
            throw new IllegalArgumentException("image exceeds 8MB limit");
        }
        if (!matches(type, data)) {
            throw new IllegalArgumentException("image bytes are not a " + type + " file");
        }
        return new Accepted(type, extension);
    }

    public static String key(UUID familyId, Long noteId, byte[] data, String extension) {
        try {
            byte[] sum = MessageDigest.getInstance("SHA-256").digest(data);
            return familyId + "/" + noteId + "/" + HexFormat.of().formatHex(sum, 0, 16) + extension;
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException(ex);
        }
    }

    private static boolean matches(String type, byte[] data) {
        return switch (type) {
            case "image/jpeg" -> startsWith(data, 0, new byte[]{(byte) 0xFF, (byte) 0xD8, (byte) 0xFF});
            case "image/png" -> startsWith(data, 0,
                    new byte[]{(byte) 0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'});
            case "image/gif" -> startsWith(data, 0, ascii("GIF87a")) || startsWith(data, 0, ascii("GIF89a"));
            case "image/webp" -> startsWith(data, 0, ascii("RIFF")) && startsWith(data, 8, ascii("WEBP"));
            default -> false;
        };
    }

    private static boolean startsWith(byte[] data, int offset, byte[] prefix) {
        if (data.length < offset + prefix.length) {
            return false;
        }
        for (int i = 0; i < prefix.length; i++) {
            if (data[offset + i] != prefix[i]) {
                return false;
            }
        }
        return true;
    }

    private static byte[] ascii(String text) {
        return text.getBytes(StandardCharsets.US_ASCII);
    }
}
