package com.example.notesjava.share.repository;

import com.example.notesjava.share.domain.Share;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Collection;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

public interface ShareRepository extends JpaRepository<Share, Long> {

    List<Share> findByFamilyIdAndNoteIdOrderByCreatedAtAscIdAsc(UUID familyId, Long noteId);

    List<Share> findByFamilyIdAndNotebookIdOrderByCreatedAtAscIdAsc(UUID familyId, Long notebookId);

    List<Share> findByFamilyIdAndNoteIdIn(UUID familyId, Collection<Long> noteIds);

    List<Share> findByFamilyIdAndNotebookIdIn(UUID familyId, Collection<Long> notebookIds);

    Optional<Share> findByIdAndFamilyId(Long id, UUID familyId);
}
