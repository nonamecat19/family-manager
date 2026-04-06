package com.example.noteapp.ui.theme

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** Darker variant of an accent color for gradient end stops. */
private fun Color.darken(factor: Float = 0.65f) = Color(
    red = (red * factor).coerceIn(0f, 1f),
    green = (green * factor).coerceIn(0f, 1f),
    blue = (blue * factor).coerceIn(0f, 1f),
    alpha = alpha,
)

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun NeuCard(
    modifier: Modifier = Modifier,
    cornerRadius: Dp = 16.dp,
    onClick: (() -> Unit)? = null,
    onLongClick: (() -> Unit)? = null,
    content: @Composable () -> Unit,
) {
    val colors = LocalAppColors.current
    val interactionModifier = when {
        onClick != null || onLongClick != null -> Modifier.combinedClickable(
            onClick = onClick ?: {},
            onLongClick = onLongClick,
        )
        else -> Modifier
    }
    val base = modifier
        .clip(RoundedCornerShape(cornerRadius))
        .background(Brush.verticalGradient(listOf(colors.surface, colors.bg)))
        .then(interactionModifier)

    Box(modifier = base) { content() }
}

@Composable
fun NeuButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false,
) {
    val colors = LocalAppColors.current
    val alpha = if (enabled) 1f else 0.38f

    Box(
        modifier = modifier
            .height(52.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(
                Brush.linearGradient(
                    listOf(colors.accent, colors.accent.darken())
                )
            )
            .clickable(enabled = enabled && !loading, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(22.dp),
                color = colors.onAccent.copy(alpha = alpha),
                strokeWidth = 2.dp,
            )
        } else {
            Text(
                text = text,
                color = colors.onAccent.copy(alpha = alpha),
                style = TextStyle(fontWeight = FontWeight.SemiBold, fontSize = 15.sp),
            )
        }
    }
}

@Composable
fun NeuTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    leadingIcon: @Composable (() -> Unit)? = null,
    trailingIcon: @Composable (() -> Unit)? = null,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    singleLine: Boolean = false,
    minLines: Int = 1,
) {
    val colors = LocalAppColors.current
    val fieldBg = Color(0xFF14141F)

    Box(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(fieldBg)
            .padding(horizontal = 16.dp, vertical = 14.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (leadingIcon != null) {
                Box(modifier = Modifier.padding(end = 12.dp)) { leadingIcon() }
            }
            Box(modifier = Modifier.weight(1f)) {
                if (value.isEmpty()) {
                    Text(
                        text = label,
                        color = colors.textSecondary,
                        style = TextStyle(fontSize = 14.sp),
                    )
                }
                BasicTextField(
                    value = value,
                    onValueChange = onValueChange,
                    textStyle = TextStyle(color = colors.textPrimary, fontSize = 14.sp),
                    cursorBrush = SolidColor(colors.accent),
                    visualTransformation = visualTransformation,
                    keyboardOptions = keyboardOptions,
                    singleLine = singleLine,
                    minLines = minLines,
                    modifier = Modifier.matchParentSize(),
                )
            }
            if (trailingIcon != null) {
                Box(modifier = Modifier.padding(start = 8.dp)) { trailingIcon() }
            }
        }
    }
}

@Composable
fun NeuTopBar(
    title: String,
    onBack: (() -> Unit)? = null,
    actions: @Composable (() -> Unit)? = null,
) {
    val colors = LocalAppColors.current

    Row(
        modifier = Modifier
            .background(colors.bg)
            .padding(horizontal = 4.dp, vertical = 8.dp)
            .padding(top = 32.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null) {
            IconButton(onClick = onBack) {
                Icon(
                    Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = "Back",
                    tint = colors.textSecondary,
                )
            }
        }
        Text(
            text = title,
            modifier = Modifier
                .weight(1f)
                .padding(horizontal = if (onBack != null) 0.dp else 16.dp),
            style = TextStyle(
                color = colors.textPrimary,
                fontWeight = FontWeight.SemiBold,
                fontSize = 18.sp,
            ),
        )
        if (actions != null) { actions() }
    }
}

@Composable
fun NeuToggle(
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    val thumbOffset by animateFloatAsState(
        targetValue = if (checked) 1f else 0f,
        animationSpec = tween(180),
        label = "thumb",
    )
    val trackColor by animateColorAsState(
        targetValue = if (checked) colors.accent.copy(alpha = 0.25f) else colors.surface,
        animationSpec = tween(180),
        label = "track",
    )
    val thumbColor by animateColorAsState(
        targetValue = if (checked) colors.accent else colors.textSecondary,
        animationSpec = tween(180),
        label = "thumb_color",
    )

    Box(
        modifier = modifier
            .width(48.dp)
            .height(26.dp)
            .clip(RoundedCornerShape(13.dp))
            .background(trackColor)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = { onCheckedChange(!checked) },
            ),
        contentAlignment = Alignment.CenterStart,
    ) {
        Box(
            modifier = Modifier
                .padding(start = 3.dp)
                .offset(x = (thumbOffset * 22).dp)
                .size(20.dp)
                .clip(CircleShape)
                .background(thumbColor),
        )
    }
}

@Composable
fun NeuSegmentedControl(
    options: List<String>,
    selected: Int,
    onSelect: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(10.dp))
            .background(colors.surface)
            .padding(3.dp),
    ) {
        options.forEachIndexed { index, label ->
            val isSelected = selected == index
            val bgColor by animateColorAsState(
                targetValue = if (isSelected) colors.accent else Color.Transparent,
                animationSpec = tween(180),
                label = "seg_bg_$index",
            )
            val textColor by animateColorAsState(
                targetValue = if (isSelected) colors.onAccent else colors.textSecondary,
                animationSpec = tween(180),
                label = "seg_txt_$index",
            )
            Box(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(8.dp))
                    .background(bgColor)
                    .clickable { onSelect(index) }
                    .padding(vertical = 9.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = label,
                    color = textColor,
                    style = TextStyle(fontWeight = FontWeight.Medium, fontSize = 13.sp),
                )
            }
        }
    }
}
