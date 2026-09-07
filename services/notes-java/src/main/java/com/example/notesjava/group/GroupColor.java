package com.example.notesjava.group;

public enum GroupColor {
    RED,
    BLUE,
    YELLOW,
    GREEN,
    PURPLE,
    ORANGE,
    BROWN;

    /** Single source for the fallback, shared by the entity default and the create request. */
    public static final GroupColor DEFAULT = BLUE;
}
