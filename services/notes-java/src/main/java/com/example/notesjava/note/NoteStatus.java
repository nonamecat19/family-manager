package com.example.notesjava.note;

public enum NoteStatus {
    ACTIVE,
    COMPLETED,
    CANCELED;

    /** Single source for the fallback, shared by the entity default and the create request. */
    public static final NoteStatus DEFAULT = ACTIVE;
}
