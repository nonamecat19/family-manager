package com.example.notesjava.share.service;

import com.example.notesjava.share.domain.Share;
import com.example.notesjava.share.repository.ShareRepository;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.function.Supplier;

@Service
public class ShareService {

    private final ShareRepository shares;

    public ShareService(ShareRepository shares) {
        this.shares = shares;
    }

    public Share shareNote(UUID familyId, Long noteId, UUID memberUserId, short permission, UUID grantedBy) {
        short subject = memberUserId == null ? Share.FAMILY : Share.MEMBER;
        return upsert(shares.findByNoteIdAndSubjectAndMemberUserId(noteId, subject, memberUserId),
                () -> Share.forNote(familyId, noteId, memberUserId, permission, grantedBy),
                permission, grantedBy);
    }

    public Share shareNotebook(UUID familyId, Long notebookId, UUID memberUserId, short permission,
                               UUID grantedBy) {
        short subject = memberUserId == null ? Share.FAMILY : Share.MEMBER;
        return upsert(shares.findByNotebookIdAndSubjectAndMemberUserId(notebookId, subject, memberUserId),
                () -> Share.forNotebook(familyId, notebookId, memberUserId, permission, grantedBy),
                permission, grantedBy);
    }

    public boolean revokeFromNote(UUID familyId, Long noteId, Long shareId) {
        return revoke(shares.findByIdAndFamilyId(shareId, familyId)
                .filter(share -> noteId.equals(share.getNoteId())));
    }

    public boolean revokeFromNotebook(UUID familyId, Long notebookId, Long shareId) {
        return revoke(shares.findByIdAndFamilyId(shareId, familyId)
                .filter(share -> notebookId.equals(share.getNotebookId())));
    }

    public List<Share> onNote(UUID familyId, Long noteId) {
        return shares.findByFamilyIdAndNoteIdOrderByCreatedAtAscIdAsc(familyId, noteId);
    }

    public List<Share> onNotebook(UUID familyId, Long notebookId) {
        return shares.findByFamilyIdAndNotebookIdOrderByCreatedAtAscIdAsc(familyId, notebookId);
    }

    private Share upsert(Optional<Share> existing, Supplier<Share> fresh, short permission,
                         UUID grantedBy) {
        if (existing.isPresent()) {
            Share share = existing.get();
            share.regrant(permission, grantedBy);
            return shares.saveAndFlush(share);
        }
        return shares.saveAndFlush(fresh.get());
    }

    private boolean revoke(Optional<Share> share) {
        share.ifPresent(shares::delete);
        return share.isPresent();
    }
}
