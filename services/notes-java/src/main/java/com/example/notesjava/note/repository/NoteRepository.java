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

import java.util.Optional;

public interface NoteRepository extends JpaRepository<Note, Long> {

    /**
     * The entity graph is what keeps the list endpoint at one query: {@code parent} and
     * {@code group} are lazy, and the response carries both ids.
     */
    @EntityGraph(attributePaths = {"parent", "group"})
    @Query("""
            SELECT n FROM Note n
            WHERE (:groupId IS NULL OR n.group.id = :groupId)
              AND (:status IS NULL OR n.status = :status)
            """)
    Page<Note> search(@Param("groupId") Long groupId,
                      @Param("status") NoteStatus status,
                      Pageable pageable);

    @EntityGraph(attributePaths = {"parent", "group"})
    Optional<Note> findWithRelationsById(Long id);

    /**
     * {@code clearAutomatically}/{@code flushAutomatically}: a bulk update bypasses the
     * persistence context, so pending changes must reach the database first and the stale
     * copies must be evicted afterwards.
     */
    @Modifying(clearAutomatically = true, flushAutomatically = true)
    @Query("UPDATE Note n SET n.group = NULL WHERE n.group.id = :groupId")
    int clearGroup(@Param("groupId") Long groupId);
}
