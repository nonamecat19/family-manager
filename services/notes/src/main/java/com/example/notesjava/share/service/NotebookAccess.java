package com.example.notesjava.share.service;

public record NotebookAccess(boolean visible, boolean canFile, boolean canManage) {

    public static final NotebookAccess HIDDEN = new NotebookAccess(false, false, false);
}
