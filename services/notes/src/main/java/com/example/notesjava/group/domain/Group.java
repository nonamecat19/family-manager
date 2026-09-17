package com.example.notesjava.group.domain;

import com.example.notesjava.common.persistence.BaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.Index;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.UUID;

@Entity
@Table(
        name = "groups",
        indexes = @Index(name = "idx_groups_family_id", columnList = "family_id")
)
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Group extends BaseEntity {

    @Column(name = "family_id", nullable = false, updatable = false)
    private UUID familyId;

    @Column(nullable = false)
    private String title;

    @Builder.Default
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 32)
    private GroupColor color = GroupColor.DEFAULT;

    public static Group of(UUID familyId, String title) {
        return Group.builder().familyId(familyId).title(title).build();
    }

    public void rename(String title, GroupColor color) {
        this.title = title;
        this.color = color;
    }
}
