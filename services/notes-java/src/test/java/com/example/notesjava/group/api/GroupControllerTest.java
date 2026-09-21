package com.example.notesjava.group.api;

import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.group.api.dto.CreateGroupRequest;
import com.example.notesjava.group.api.dto.GroupResponse;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.domain.GroupColor;
import com.example.notesjava.group.service.GroupService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import tools.jackson.databind.ObjectMapper;

import java.time.Instant;

import static org.hamcrest.Matchers.endsWith;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(GroupController.class)
class GroupControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockitoBean
    private GroupService groupService;

    @Test
    void createReturns201WithALocationHeader() throws Exception {
        when(groupService.create(any())).thenReturn(response(3L, "work"));

        mockMvc.perform(post("/api/groups").contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(new CreateGroupRequest("work", GroupColor.RED))))
                .andExpect(status().isCreated())
                .andExpect(header().string("Location", endsWith("/api/groups/3")));
    }

    @Test
    void rejectsABlankTitle() throws Exception {
        mockMvc.perform(post("/api/groups").contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(new CreateGroupRequest(" ", null))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors.title").exists());
    }

    @Test
    void rejectsAnUpdateWithoutAColour() throws Exception {
        mockMvc.perform(put("/api/groups/1").contentType(MediaType.APPLICATION_JSON)
                        .content("{\"title\":\"work\"}"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors.color").exists());
    }

    @Test
    void rejectsAnUnknownColourWithA400RatherThanA500() throws Exception {
        mockMvc.perform(post("/api/groups").contentType(MediaType.APPLICATION_JSON)
                        .content("{\"title\":\"work\",\"color\":\"CHARTREUSE\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void missingGroupBecomesA404ProblemDetail() throws Exception {
        when(groupService.getById(404L)).thenThrow(new ResourceNotFoundException("Group", 404L));

        mockMvc.perform(get("/api/groups/404"))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.detail").value("Group 404 not found"));
    }

    private static GroupResponse response(Long id, String title) {
        Instant now = Instant.now();
        return new GroupResponse(id, title, GroupColor.RED, 0L, now, now);
    }
}
