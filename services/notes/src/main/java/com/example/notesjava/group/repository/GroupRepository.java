package com.example.notesjava.group.repository;

import com.example.notesjava.group.domain.Group;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;
import java.util.UUID;

public interface GroupRepository extends JpaRepository<Group, Long> {

    Page<Group> findAllByFamilyId(UUID familyId, Pageable pageable);

    Optional<Group> findByIdAndFamilyId(Long id, UUID familyId);
}
