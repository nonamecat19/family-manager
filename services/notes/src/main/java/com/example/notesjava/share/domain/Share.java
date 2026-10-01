package com.example.notesjava.share.domain;

import com.example.notesjava.common.persistence.BaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.Comparator;
import java.util.UUID;

@Entity
@Table(name = "shares")
@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
public class Share extends BaseEntity {

    public static final short MEMBER = 1;
    public static final short FAMILY = 2;
    public static final short VIEW = 1;
    public static final short EDIT = 2;

    public static final Comparator<Share> ORDER = Comparator
            .comparing(Share::getCreatedAt, Comparator.nullsLast(Comparator.naturalOrder()))
            .thenComparing(Share::getId, Comparator.nullsLast(Comparator.naturalOrder()));

    @Column(name = "family_id", nullable = false, updatable = false)
    private UUID familyId;

    @Column(name = "note_id", updatable = false)
    private Long noteId;

    @Column(name = "notebook_id", updatable = false)
    private Long notebookId;

    @Column(nullable = false, updatable = false)
    private short subject;

    @Column(name = "member_user_id", updatable = false)
    private UUID memberUserId;

    @Column(nullable = false)
    private short permission;

    @Column(name = "granted_by_user_id", nullable = false)
    private UUID grantedByUserId;

    public static Share forNote(UUID familyId, Long noteId, UUID memberUserId, short permission, UUID grantedBy) {
        Share share = grant(familyId, memberUserId, permission, grantedBy);
        share.noteId = noteId;
        return share;
    }

    public static Share forNotebook(UUID familyId, Long notebookId, UUID memberUserId, short permission,
                                    UUID grantedBy) {
        Share share = grant(familyId, memberUserId, permission, grantedBy);
        share.notebookId = notebookId;
        return share;
    }

    private static Share grant(UUID familyId, UUID memberUserId, short permission, UUID grantedBy) {
        Share share = new Share();
        share.familyId = familyId;
        share.subject = memberUserId == null ? FAMILY : MEMBER;
        share.memberUserId = memberUserId;
        share.permission = permission;
        share.grantedByUserId = grantedBy;
        return share;
    }

    public void regrant(short permission, UUID grantedBy) {
        this.permission = permission;
        this.grantedByUserId = grantedBy;
    }

    public boolean reaches(UUID userId) {
        return subject == FAMILY || (userId != null && userId.equals(memberUserId));
    }

    public boolean allowsEdit() {
        return permission == EDIT;
    }
}
