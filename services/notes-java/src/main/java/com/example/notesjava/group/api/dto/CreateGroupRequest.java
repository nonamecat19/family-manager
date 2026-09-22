package com.example.notesjava.group.api.dto;

import com.example.notesjava.group.domain.GroupColor;
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
