package com.example.notesjava.note;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

/**
 * A full replacement: a PUT that omitted {@code status} or {@code priority} would otherwise
 * silently reset them, so both are required. A null {@code parentId}/{@code groupId} is a
 * meaningful value — it unfiles the note.
 */
public record UpdateNoteRequest(
        @NotBlank @Size(max = 255) String title,
        @Size(max = 10_000) String content,
        @NotNull NoteStatus status,
        @NotNull NotePriority priority,
        Long parentId,
        Long groupId
) {
}
