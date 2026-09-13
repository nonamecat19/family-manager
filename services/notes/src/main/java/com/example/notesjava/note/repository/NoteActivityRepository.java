package com.example.notesjava.note.repository;

import com.example.notesjava.note.domain.NoteActivity;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

import java.util.List;
import java.util.UUID;

public interface NoteActivityRepository extends JpaRepository<NoteActivity, Long> {

    @Query("""
            SELECT a FROM NoteActivity a
            WHERE a.familyId = :familyId
              AND a.noteId = :noteId
            ORDER BY a.createdAt DESC, a.id DESC
            """)
    List<NoteActivity> findForNote(@Param("familyId") UUID familyId,
                                   @Param("noteId") Long noteId,
                                   Pageable pageable);
}
