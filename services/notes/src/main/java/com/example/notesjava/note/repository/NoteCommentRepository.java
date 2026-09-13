package com.example.notesjava.note.repository;

import com.example.notesjava.note.domain.NoteComment;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

public interface NoteCommentRepository extends JpaRepository<NoteComment, Long> {

    @Query("""
            SELECT c FROM NoteComment c
            WHERE c.familyId = :familyId
              AND c.noteId = :noteId
              AND (:includeResolved = TRUE OR c.resolved = FALSE)
            ORDER BY c.createdAt, c.id
            """)
    List<NoteComment> findForNote(@Param("familyId") UUID familyId,
                                  @Param("noteId") Long noteId,
                                  @Param("includeResolved") boolean includeResolved);

    Optional<NoteComment> findByIdAndFamilyId(Long id, UUID familyId);
}
