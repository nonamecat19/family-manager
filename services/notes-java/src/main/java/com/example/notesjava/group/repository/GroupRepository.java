package com.example.notesjava.group.repository;

import com.example.notesjava.group.domain.Group;
import org.springframework.data.jpa.repository.JpaRepository;

public interface GroupRepository extends JpaRepository<Group, Long> {
}
