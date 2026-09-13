package com.example.notesjava.group.api.dto;

import com.example.notesjava.group.domain.GroupColor;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

/**
 * A full replacement, so every field is required — a PUT that left {@code color} out would
 * otherwise silently reset it.
 */
public record UpdateGroupRequest(
        @NotBlank @Size(max = 255) String title,
        @NotNull GroupColor color
) {
}
