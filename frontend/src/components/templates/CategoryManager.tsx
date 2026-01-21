import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { templateService } from '../../services/templateService';
import type { TemplateCategory } from '../../types/template';

export const CategoryManager: React.FC = () => {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [editingCategory, setEditingCategory] =
    useState<Partial<TemplateCategory> | null>(null);

  // Fetch categories
  const { data: categories, isLoading } = useQuery({
    queryKey: ['template-categories'],
    queryFn: () => templateService.listCategories(),
  });

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (category: Partial<TemplateCategory>) =>
      templateService.createCategory(category),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['template-categories'] });
      setIsEditing(false);
      setEditingCategory(null);
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({
      id,
      updates,
    }: {
      id: string;
      updates: Partial<TemplateCategory>;
    }) => templateService.updateCategory(id, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['template-categories'] });
      setIsEditing(false);
      setEditingCategory(null);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => templateService.deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['template-categories'] });
    },
  });

  const handleCreate = () => {
    setEditingCategory({
      name: '',
      description: '',
      icon: '📁',
      color: '#3B82F6',
      sort_order: categories ? categories.length : 0,
      is_active: true,
    });
    setIsEditing(true);
  };

  const handleEdit = (category: TemplateCategory) => {
    setEditingCategory({ ...category });
    setIsEditing(true);
  };

  const handleSave = () => {
    if (!editingCategory) return;

    if (editingCategory.id) {
      updateMutation.mutate({
        id: editingCategory.id,
        updates: editingCategory,
      });
    } else {
      createMutation.mutate(editingCategory);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setEditingCategory(null);
  };

  const handleDelete = (category: TemplateCategory) => {
    if (
      window.confirm(
        `Are you sure you want to delete the category "${category.name}"?`
      )
    ) {
      deleteMutation.mutate(category.id);
    }
  };

  const commonIcons = [
    '📁',
    '🛠️',
    '💼',
    '🏗️',
    '⚡',
    '🎨',
    '📊',
    '🔧',
    '📦',
    '🌟',
  ];
  const commonColors = [
    '#3B82F6', // Blue
    '#10B981', // Green
    '#F59E0B', // Amber
    '#EF4444', // Red
    '#8B5CF6', // Purple
    '#EC4899', // Pink
    '#14B8A6', // Teal
    '#F97316', // Orange
  ];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-gray-600">Loading categories...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">
            Template Categories
          </h2>
          <p className="mt-1 text-sm text-gray-500">
            Organize your templates into categories for easier management
          </p>
        </div>
        <button
          type="button"
          onClick={handleCreate}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Add Category
        </button>
      </div>

      {/* Categories List */}
      <div className="space-y-3">
        {categories && categories.length > 0 ? (
          categories.map((category) => (
            <div
              key={category.id}
              className="flex items-center justify-between rounded-lg border bg-white p-4"
            >
              <div className="flex items-center gap-4">
                <div
                  className="flex h-12 w-12 items-center justify-center rounded-lg text-2xl"
                  style={{ backgroundColor: `${category.color}20` }}
                >
                  {category.icon}
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">
                    {category.name}
                  </h3>
                  <p className="text-sm text-gray-500">
                    {category.description}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-500">
                  Order: {category.sort_order}
                </span>
                <button
                  type="button"
                  onClick={() => handleEdit(category)}
                  className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(category)}
                  disabled={deleteMutation.isPending}
                  className="rounded-md border border-red-300 bg-white px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        ) : (
          <div className="rounded-lg border bg-white p-8 text-center">
            <svg
              className="mx-auto h-12 w-12 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
              />
            </svg>
            <h3 className="mt-2 text-sm font-medium text-gray-900">
              No categories
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Get started by creating a new category
            </p>
          </div>
        )}
      </div>

      {/* Edit Dialog */}
      {isEditing && editingCategory && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold text-gray-900">
              {editingCategory.id ? 'Edit Category' : 'Create Category'}
            </h3>

            <div className="mt-4 space-y-4">
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editingCategory.name || ''}
                  onChange={(e) =>
                    setEditingCategory({
                      ...editingCategory,
                      name: e.target.value,
                    })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="e.g., Service, Product, Consulting"
                />
              </div>

              {/* Description */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Description
                </label>
                <textarea
                  value={editingCategory.description || ''}
                  onChange={(e) =>
                    setEditingCategory({
                      ...editingCategory,
                      description: e.target.value,
                    })
                  }
                  rows={3}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="Brief description of this category"
                />
              </div>

              {/* Icon */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Icon
                </label>
                <div className="mt-2 grid grid-cols-5 gap-2">
                  {commonIcons.map((icon) => (
                    <button
                      key={icon}
                      type="button"
                      onClick={() =>
                        setEditingCategory({ ...editingCategory, icon })
                      }
                      className={`flex h-12 w-12 items-center justify-center rounded-lg border-2 text-2xl transition-all ${
                        editingCategory.icon === icon
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      {icon}
                    </button>
                  ))}
                </div>
                <input
                  type="text"
                  value={editingCategory.icon || ''}
                  onChange={(e) =>
                    setEditingCategory({
                      ...editingCategory,
                      icon: e.target.value,
                    })
                  }
                  className="mt-2 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="Or enter custom emoji"
                  maxLength={2}
                />
              </div>

              {/* Color */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Color
                </label>
                <div className="mt-2 grid grid-cols-4 gap-2">
                  {commonColors.map((color) => (
                    <button
                      key={color}
                      type="button"
                      onClick={() =>
                        setEditingCategory({ ...editingCategory, color })
                      }
                      className={`h-10 w-full rounded-lg border-2 ${
                        editingCategory.color === color
                          ? 'border-gray-900 ring-2 ring-gray-900 ring-offset-2'
                          : 'border-gray-200'
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
                <input
                  type="text"
                  value={editingCategory.color || ''}
                  onChange={(e) =>
                    setEditingCategory({
                      ...editingCategory,
                      color: e.target.value,
                    })
                  }
                  className="mt-2 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm"
                  placeholder="#3B82F6"
                />
              </div>

              {/* Sort Order */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Sort Order
                </label>
                <input
                  type="number"
                  value={editingCategory.sort_order || 0}
                  onChange={(e) =>
                    setEditingCategory({
                      ...editingCategory,
                      sort_order: parseInt(e.target.value) || 0,
                    })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  min="0"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Lower numbers appear first in the list
                </p>
              </div>

              {/* Preview */}
              <div className="rounded-md bg-gray-50 p-4">
                <p className="mb-2 text-xs font-medium text-gray-700">
                  Preview
                </p>
                <div className="flex items-center gap-3">
                  <div
                    className="flex h-12 w-12 items-center justify-center rounded-lg text-2xl"
                    style={{ backgroundColor: `${editingCategory.color}20` }}
                  >
                    {editingCategory.icon}
                  </div>
                  <div>
                    <p className="font-semibold text-gray-900">
                      {editingCategory.name || 'Category Name'}
                    </p>
                    <p className="text-sm text-gray-500">
                      {editingCategory.description || 'Category description'}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Actions */}
            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={handleCancel}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={
                  !editingCategory.name ||
                  createMutation.isPending ||
                  updateMutation.isPending
                }
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {createMutation.isPending || updateMutation.isPending
                  ? 'Saving...'
                  : editingCategory.id
                    ? 'Update Category'
                    : 'Create Category'}
              </button>
            </div>

            {/* Error Display */}
            {(createMutation.isError || updateMutation.isError) && (
              <div className="mt-4 rounded-md bg-red-50 p-4">
                <p className="text-sm text-red-800">
                  {(createMutation.error || updateMutation.error) instanceof
                  Error
                    ? (createMutation.error || updateMutation.error)?.message
                    : 'An error occurred'}
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
