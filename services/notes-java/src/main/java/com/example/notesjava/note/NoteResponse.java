package com.example.notesjava.note;

import com.example.notesjava.group.Group;

import java.time.Instant;

public record NoteResponse(
        Long id,
        String title,
        String content,
        NoteStatus status,
        NotePriority priority,
        Long parentId,
        Long groupId,
        Long version,
        Instant createdAt,
        Instant updatedAt
) {
    static NoteResponse from(Note note) {
        Note parent = note.getParent();
        Group group = note.getGroup();
        return new NoteResponse(
                note.getId(),
                note.getTitle(),
                note.getContent(),
                note.getStatus(),
                note.getPriority(),
                parent == null ? null : parent.getId(),
                group == null ? null : group.getId(),
                note.getVersion(),
                note.getCreatedAt(),
                note.getUpdatedAt()
        );
    }
}
