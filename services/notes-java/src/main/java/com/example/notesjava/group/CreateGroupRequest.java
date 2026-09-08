package com.example.notesjava.group;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record CreateGroupRequest(
        @NotBlank @Size(max = 255) String title,
        GroupColor color
) {
    public CreateGroupRequest {
        color = color == null ? GroupColor.DEFAULT : color;
    }
}
