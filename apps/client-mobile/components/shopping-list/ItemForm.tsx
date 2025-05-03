import React, {useEffect, useState} from 'react';
import {ScrollView, StyleSheet, View} from 'react-native';
import {Button, IndexPath, Input, SelectItem, Text} from '@ui-kitten/components';
import {Category, ListItem, Tag} from '@/lib/models';

interface ItemFormProps {
    initialValues?: Partial<ListItem>;
    categories: Category[];
    tags: Tag[];
    onSubmit: (values: Omit<ListItem, 'id'>) => void;
    onCancel: () => void;
    isSubmitting: boolean;
}

export const ItemForm = ({
                             initialValues,
                             categories,
                             tags,
                             onSubmit,
                             onCancel,
                             isSubmitting,
                         }: ItemFormProps) => {
    const [name, setName] = useState(initialValues?.title || '');
    const [selectedCategoryIndex, setSelectedCategoryIndex] = useState<IndexPath | null>(null);
    const [selectedTagIndices, setSelectedTagIndices] = useState<IndexPath[]>([]);
    const [errors, setErrors] = useState<{ name?: string }>({});

    useEffect(() => {
        if (initialValues?.categoryId && categories.length > 0) {
            const categoryIndex = categories.findIndex(c => c.id === initialValues.categoryId);
            if (categoryIndex !== -1) {
                setSelectedCategoryIndex(new IndexPath(categoryIndex));
            }
        }

        if (initialValues?.tagIds && initialValues.tagIds.length > 0 && tags.length > 0) {
            const tagIndices = initialValues.tagIds
                .map(tagId => tags.findIndex(t => t.id === tagId))
                .filter(index => index !== -1)
                .map(index => new IndexPath(index));

            if (tagIndices.length > 0) {
                setSelectedTagIndices(tagIndices);
            }
        }
    }, [initialValues, categories, tags]);

    const handleSubmit = () => {
        const newErrors: { name?: string } = {};

        if (!name.trim()) {
            newErrors.name = 'Name is required';
        }

        if (Object.keys(newErrors).length > 0) {
            setErrors(newErrors);
            return;
        }

        const categoryId = selectedCategoryIndex
            ? categories[selectedCategoryIndex.row].id
            : undefined;

        const tagIds = selectedTagIndices.length > 0
            ? selectedTagIndices.map(index => tags[index.row].id)
            : undefined;

        onSubmit({
            title: name.trim(),
            categoryId,
            tagIds,
        });
    };

    const renderCategoryOption = (category: Category) => (
        <SelectItem key={category.id} title={category.name}/>
    );

    const renderTagOption = (tag: Tag) => (
        <SelectItem key={tag.id} title={tag.name}/>
    );

    const displayCategoryValue = () => {
        if (selectedCategoryIndex !== null && categories.length > 0) {
            return categories[selectedCategoryIndex.row].name;
        }
        return 'Select category';
    };

    const displayTagsValue = () => {
        if (selectedTagIndices.length > 0 && tags.length > 0) {
            return selectedTagIndices
                .map(index => tags[index.row].name)
                .join(', ');
        }
        return 'Select tags';
    };

    return (
        <ScrollView style={styles.container}>
            <Text category="h6" style={styles.title}>
                {initialValues?.id ? 'Edit Item' : 'Add New Item'}
            </Text>

            <View style={styles.formGroup}>
                <Text style={styles.label}>Name</Text>
                <Input
                    placeholder="Enter item name"
                    value={name}
                    onChangeText={setName}
                    status={errors.name ? 'danger' : 'basic'}
                    caption={errors.name}
                />
            </View>

            <View style={styles.formGroup}>
                <Text style={styles.label}>Category (optional)</Text>
                {/*  <Select*/}
                {/*    placeholder="Select category"*/}
                {/*    value={displayCategoryValue()}*/}
                {/*    // selectedIndex={selectedCategoryIndex}*/}
                {/*    onSelect={index => setSelectedCategoryIndex(index as IndexPath)}*/}
                {/*  >*/}
                {/*    {categories.map(renderCategoryOption)}*/}
                {/*  </Select>*/}
            </View>

            <View style={styles.formGroup}>
                <Text style={styles.label}>Tags (optional)</Text>
                {/*  <Select*/}
                {/*    placeholder="Select tags"*/}
                {/*    value={displayTagsValue()}*/}
                {/*    selectedIndex={selectedTagIndices}*/}
                {/*    onSelect={index => setSelectedTagIndices(index as IndexPath[])}*/}
                {/*    multiSelect*/}
                {/*  >*/}
                {/*    {tags.map(renderTagOption)}*/}
                {/*  </Select>*/}
            </View>

            <View style={styles.buttonGroup}>
                <Button
                    appearance="outline"
                    status="basic"
                    onPress={onCancel}
                    style={styles.button}
                    disabled={isSubmitting}
                >
                    Cancel
                </Button>
                <Button
                    onPress={handleSubmit}
                    style={styles.button}
                    disabled={isSubmitting}
                >
                    {isSubmitting ? 'Saving...' : 'Save'}
                </Button>
            </View>
        </ScrollView>
    );
};

const styles = StyleSheet.create({
    container: {
        padding: 16,
    },
    title: {
        marginBottom: 16,
    },
    formGroup: {
        marginBottom: 16,
    },
    label: {
        marginBottom: 4,
        fontWeight: '500',
    },
    buttonGroup: {
        flexDirection: 'row',
        justifyContent: 'flex-end',
        marginTop: 8,
    },
    button: {
        marginLeft: 8,
    },
});