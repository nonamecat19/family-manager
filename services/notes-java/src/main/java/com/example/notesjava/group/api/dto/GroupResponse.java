package com.example.notesjava.group.api.dto;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.domain.GroupColor;
import java.time.Instant;

public record GroupResponse(
        Long id,
        String title,
        GroupColor color,
        Long version,
        Instant createdAt,
        Instant updatedAt
) {
    public static GroupResponse from(Group group) {
        return new GroupResponse(
                group.getId(),
                group.getTitle(),
                group.getColor(),
                group.getVersion(),
                group.getCreatedAt(),
                group.getUpdatedAt()
        );
    }
}
