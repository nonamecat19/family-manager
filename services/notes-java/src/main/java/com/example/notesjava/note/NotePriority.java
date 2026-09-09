package com.example.notesjava.note;

public enum NotePriority {
    LOW,
    NORMAL,
    HIGH;

    /** Single source for the fallback, shared by the entity default and the create request. */
    public static final NotePriority DEFAULT = NORMAL;
}
