package com.example.notesjava.note.api.dto;


import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NotePriority;
import com.example.notesjava.note.domain.NoteStatus;
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
    public static NoteResponse from(Note note) {
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
