package com.example.notesjava.note.repository;

import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NoteStatus;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.EntityGraph;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

public interface NoteRepository extends JpaRepository<Note, Long> {

    @EntityGraph(attributePaths = {"parent", "group"})
    @Query("""
            SELECT n FROM Note n
            WHERE n.familyId = :familyId
              AND (:groupId IS NULL OR n.group.id = :groupId)
              AND (:status IS NULL OR n.status = :status)
            """)
    Page<Note> search(@Param("familyId") UUID familyId,
                      @Param("groupId") Long groupId,
                      @Param("status") NoteStatus status,
                      Pageable pageable);

    @EntityGraph(attributePaths = {"parent", "group"})
    Optional<Note> findWithRelationsByIdAndFamilyId(Long id, UUID familyId);

    Optional<Note> findByIdAndFamilyId(Long id, UUID familyId);

    @EntityGraph(attributePaths = {"group"})
    @Query("""
            SELECT n FROM Note n
            WHERE n.familyId = :familyId
              AND (:groupId IS NULL OR n.group.id = :groupId)
              AND (:starredOnly = FALSE OR n.starred = TRUE)
              AND (:includeArchived = TRUE OR n.archived = FALSE)
              AND (:archivedOnly = FALSE OR n.archived = TRUE)
            """)
    Page<Note> listForContract(@Param("familyId") UUID familyId,
                               @Param("groupId") Long groupId,
                               @Param("starredOnly") boolean starredOnly,
                               @Param("includeArchived") boolean includeArchived,
                               @Param("archivedOnly") boolean archivedOnly,
                               Pageable pageable);

    @EntityGraph(attributePaths = {"group"})
    @Query("""
            SELECT n FROM Note n
            WHERE n.familyId = :familyId
              AND n.archived = FALSE
              AND (LOWER(n.title) LIKE :needle OR LOWER(COALESCE(n.content, '')) LIKE :needle)
            ORDER BY n.updatedAt DESC
            """)
    List<Note> searchText(@Param("familyId") UUID familyId, @Param("needle") String needle, Pageable pageable);

    long countByFamilyIdAndGroupId(UUID familyId, Long groupId);

    @Modifying(clearAutomatically = true, flushAutomatically = true)
    @Query("UPDATE Note n SET n.group = NULL WHERE n.group.id = :groupId AND n.familyId = :familyId")
    int clearGroup(@Param("groupId") Long groupId, @Param("familyId") UUID familyId);
}
