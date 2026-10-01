package com.example.notesjava.share.repository;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.share.domain.Share;
import jakarta.persistence.criteria.CriteriaBuilder;
import jakarta.persistence.criteria.CriteriaQuery;
import jakarta.persistence.criteria.Expression;
import jakarta.persistence.criteria.Path;
import jakarta.persistence.criteria.Predicate;
import jakarta.persistence.criteria.Root;
import jakarta.persistence.criteria.Subquery;
import org.springframework.data.jpa.domain.Specification;

import java.util.UUID;

public final class AccessSpecifications {

    private AccessSpecifications() {
    }

    public static Specification<Note> visibleNotes(UUID familyId, UUID userId) {
        return (root, query, builder) -> builder.and(
                builder.equal(root.get("familyId"), familyId),
                builder.or(
                        builder.isNull(root.get("ownerUserId")),
                        owns(builder, root.get("ownerUserId"), userId),
                        builder.exists(directShare(root, query, builder, userId)),
                        builder.exists(notebookShare(root, query, builder, userId))));
    }

    public static Specification<Note> sharedWithMe(UUID familyId, UUID userId) {
        return visibleNotes(familyId, userId).and((root, query, builder) -> builder.and(
                builder.isNotNull(root.get("ownerUserId")),
                builder.not(owns(builder, root.get("ownerUserId"), userId))));
    }

    public static Specification<Group> visibleNotebooks(UUID familyId, UUID userId) {
        return (root, query, builder) -> builder.and(
                builder.equal(root.get("familyId"), familyId),
                builder.or(
                        builder.isNull(root.get("ownerUserId")),
                        owns(builder, root.get("ownerUserId"), userId),
                        builder.exists(groupShare(root, query, builder, userId))));
    }

    public static Specification<Group> notebooksSharedWithMe(UUID familyId, UUID userId) {
        return (root, query, builder) -> builder.and(
                builder.equal(root.get("familyId"), familyId),
                builder.isNotNull(root.get("ownerUserId")),
                builder.not(owns(builder, root.get("ownerUserId"), userId)),
                builder.exists(groupShare(root, query, builder, userId)));
    }

    private static Subquery<Long> directShare(Root<Note> note, CriteriaQuery<?> query, CriteriaBuilder builder,
                                              UUID userId) {
        Subquery<Long> subquery = query.subquery(Long.class);
        Root<Share> share = subquery.from(Share.class);
        return subquery.select(share.get("id")).where(
                builder.equal(share.get("noteId"), note.get("id")),
                reaches(builder, share, userId));
    }

    private static Subquery<Long> notebookShare(Root<Note> note, CriteriaQuery<?> query, CriteriaBuilder builder,
                                                UUID userId) {
        Subquery<Long> subquery = query.subquery(Long.class);
        Root<Share> share = subquery.from(Share.class);
        Root<Group> notebook = subquery.from(Group.class);
        return subquery.select(share.get("id")).where(
                builder.equal(notebook.get("id"), share.get("notebookId")),
                builder.equal(share.get("notebookId"), note.get("group").get("id")),
                builder.equal(notebook.get("ownerUserId"), note.get("ownerUserId")),
                reaches(builder, share, userId));
    }

    private static Subquery<Long> groupShare(Root<Group> notebook, CriteriaQuery<?> query, CriteriaBuilder builder,
                                             UUID userId) {
        Subquery<Long> subquery = query.subquery(Long.class);
        Root<Share> share = subquery.from(Share.class);
        return subquery.select(share.get("id")).where(
                builder.equal(share.get("notebookId"), notebook.get("id")),
                reaches(builder, share, userId));
    }

    private static Predicate reaches(CriteriaBuilder builder, Path<Share> share, UUID userId) {
        Predicate family = builder.equal(share.get("subject"), Share.FAMILY);
        if (userId == null) {
            return family;
        }
        return builder.or(family, builder.equal(share.get("memberUserId"), userId));
    }

    private static Predicate owns(CriteriaBuilder builder, Expression<UUID> owner, UUID userId) {
        return userId == null ? builder.disjunction() : builder.equal(owner, userId);
    }
}
