package com.example.notesjava.note.api.dto;

import com.example.notesjava.note.domain.NotePriority;
import com.example.notesjava.note.domain.NoteStatus;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

public record UpdateNoteRequest(
        @NotBlank @Size(max = 255) String title,
        @Size(max = 10_000) String content,
        @NotNull NoteStatus status,
        @NotNull NotePriority priority,
        Long parentId,
        Long groupId
) {
}
