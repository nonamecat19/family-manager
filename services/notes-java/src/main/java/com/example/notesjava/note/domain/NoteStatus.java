package com.example.notesjava.note.domain;

public enum NoteStatus {
    ACTIVE,
    COMPLETED,
    CANCELED;

    public static final NoteStatus DEFAULT = ACTIVE;
}
