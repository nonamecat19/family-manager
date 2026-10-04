package com.example.notesjava.share.service;

import com.example.notesjava.share.domain.Share;
import com.example.notesjava.share.repository.ShareRepository;
import jakarta.persistence.EntityManager;
import org.springframework.jdbc.core.namedparam.MapSqlParameterSource;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Service;

import java.sql.Types;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Service
public class ShareService {

    private final ShareRepository shares;
    private final NamedParameterJdbcTemplate jdbc;
    private final EntityManager entities;

    public ShareService(ShareRepository shares, NamedParameterJdbcTemplate jdbc, EntityManager entities) {
        this.shares = shares;
        this.jdbc = jdbc;
        this.entities = entities;
    }

    public Share shareNote(UUID familyId, Long noteId, UUID memberUserId, short permission, UUID grantedBy) {
        return upsert("note_id", familyId, noteId, memberUserId, permission, grantedBy);
    }

    public Share shareNotebook(UUID familyId, Long notebookId, UUID memberUserId, short permission,
                               UUID grantedBy) {
        return upsert("notebook_id", familyId, notebookId, memberUserId, permission, grantedBy);
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

    private Share upsert(String target, UUID familyId, Long targetId, UUID memberUserId, short permission,
                         UUID grantedBy) {
        boolean member = memberUserId != null;
        String conflict = member
                ? "(" + target + ", member_user_id) WHERE " + target + " IS NOT NULL AND subject = " + Share.MEMBER
                : "(" + target + ") WHERE " + target + " IS NOT NULL AND subject = " + Share.FAMILY;
        String sql = "INSERT INTO shares (family_id, " + target + ", subject, member_user_id, permission,"
                + " granted_by_user_id, version, created_at, updated_at)"
                + " VALUES (:familyId, :targetId, :subject, :memberUserId, :permission, :grantedBy, 0, now(), now())"
                + " ON CONFLICT " + conflict
                + " DO UPDATE SET permission = EXCLUDED.permission,"
                + " granted_by_user_id = EXCLUDED.granted_by_user_id,"
                + " version = shares.version + 1, updated_at = now()"
                + " RETURNING id";
        MapSqlParameterSource params = new MapSqlParameterSource()
                .addValue("familyId", familyId, Types.OTHER)
                .addValue("targetId", targetId, Types.BIGINT)
                .addValue("subject", member ? Share.MEMBER : Share.FAMILY, Types.SMALLINT)
                .addValue("memberUserId", memberUserId, Types.OTHER)
                .addValue("permission", permission, Types.SMALLINT)
                .addValue("grantedBy", grantedBy, Types.OTHER);
        Long id = jdbc.queryForObject(sql, params, Long.class);
        Share share = shares.findById(id).orElseThrow();
        entities.refresh(share);
        return share;
    }

    private boolean revoke(Optional<Share> share) {
        share.ifPresent(shares::delete);
        return share.isPresent();
    }
}
