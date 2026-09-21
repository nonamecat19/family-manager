package com.example.notesjava.group.domain;

import com.example.notesjava.common.persistence.BaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Entity
@Table(name = "groups")
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Group extends BaseEntity {

    @Column(nullable = false)
    private String title;

    @Builder.Default
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 32)
    private GroupColor color = GroupColor.DEFAULT;

    public static Group of(String title) {
        return Group.builder().title(title).build();
    }

    public void rename(String title, GroupColor color) {
        this.title = title;
        this.color = color;
    }
}
