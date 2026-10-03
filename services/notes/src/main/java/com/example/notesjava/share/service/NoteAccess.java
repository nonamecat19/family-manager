package com.example.notesjava.share.service;

import com.example.notesjava.share.domain.Share;

import java.util.List;

public record NoteAccess(boolean visible, boolean canEdit, boolean canManage, boolean shared, List<Share> shares) {

    public static final NoteAccess HIDDEN = new NoteAccess(false, false, false, false, List.of());
}
