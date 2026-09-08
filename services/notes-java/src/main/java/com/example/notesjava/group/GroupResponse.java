package com.example.notesjava.group;

import java.time.Instant;

public record GroupResponse(
        Long id,
        String title,
        GroupColor color,
        Long version,
        Instant createdAt,
        Instant updatedAt
) {
    static GroupResponse from(Group group) {
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
