package com.example.notesjava.note;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record CreateNoteRequest(
        @NotBlank @Size(max = 255) String title,
        @Size(max = 10_000) String content,
        NoteStatus status,
        NotePriority priority,
        Long parentId,
        Long groupId
) {
    public CreateNoteRequest {
        status = status == null ? NoteStatus.DEFAULT : status;
        priority = priority == null ? NotePriority.DEFAULT : priority;
    }
}
