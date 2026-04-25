"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { ArrowLeft, Loader2, Pencil } from "lucide-react";
import Button from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Modal } from "@/components/ui/Modal";
import { FormField, Input, Select, Textarea } from "@/components/ui/FormField";
import { useProduct, useUpdateProduct, useDeleteProduct } from "@/lib/use-products";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";

const CATEGORY_OPTIONS = [
  { value: "", label: "No category" },
  { value: "Electronics", label: "Electronics" },
  { value: "Furniture", label: "Furniture" },
  { value: "Clothing", label: "Clothing" },
  { value: "Food & Beverage", label: "Food & Beverage" },
  { value: "Tools", label: "Tools" },
];

interface EditFormValues {
  name: string;
  description: string;
  category: string;
  unit_price: string;
}

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading } = useProduct(id);
  const updateProduct = useUpdateProduct();
  const deleteProduct = useDeleteProduct();

  const [editModalOpen, setEditModalOpen] = useState(false);

  const product = data?.data;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<EditFormValues>();

  const openEdit = () => {
    if (!product) return;
    reset({
      name: product.name,
      description: product.description || "",
      category: product.category || "",
      unit_price: String(product.unit_price),
    });
    setEditModalOpen(true);
  };

  const onSubmit = (data: EditFormValues) => {
    updateProduct.mutate(
      {
        id,
        name: data.name,
        description: data.description || undefined,
        category: data.category || undefined,
        unit_price: parseFloat(data.unit_price),
      },
      {
        onSuccess: () => {
          toastSuccess("Product updated");
          setEditModalOpen(false);
          queryClient.invalidateQueries({ queryKey: ["products", id] });
        },
        onError: toastError,
      },
    );
  };

  const handleDelete = () => {
    deleteProduct.mutate(
      { id },
      {
        onSuccess: () => {
          toastSuccess("Product deleted");
          router.push("/products");
        },
        onError: toastError,
      },
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!product) {
    return (
      <div className="text-center py-20 text-gray-500">Product not found</div>
    );
  }

  return (
    <div className="space-y-6">
      {}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/products")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-gray-900">{product.name}</h1>
          <p className="mt-0.5 font-mono text-sm text-gray-500">{product.sku}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={openEdit}>
            <Pencil className="h-4 w-4" />
            Edit
          </Button>
          <Button
            variant="danger"
            onClick={handleDelete}
            loading={deleteProduct.isPending}
          >
            Delete
          </Button>
        </div>
      </div>

      {}
      <div className="grid gap-6 md:grid-cols-4">
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">SKU</h2>
          <p className="font-mono text-lg font-medium text-gray-900">
            {product.sku}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Category</h2>
          <p className="text-lg font-medium text-gray-900">
            {product.category || "-"}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">
            Unit Price
          </h2>
          <p className="text-lg font-medium text-gray-900">
            ${product.unit_price.toFixed(2)}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Created</h2>
          <p className="text-lg font-medium text-gray-900">
            {new Date(product.created_at).toLocaleDateString()}
          </p>
        </Card>
      </div>

      {}
      {product.description && (
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">
            Description
          </h2>
          <p className="text-sm text-gray-600">{product.description}</p>
        </Card>
      )}

      {}
      <Modal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        title="Edit Product"
        size="md"
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FormField label="Name" required error={errors.name}>
            <Input
              registration={register("name", { required: "Name is required" })}
              error={errors.name}
            />
          </FormField>
          <FormField label="Category">
            <Select
              registration={register("category")}
              options={CATEGORY_OPTIONS}
            />
          </FormField>
          <FormField label="Unit Price" required error={errors.unit_price}>
            <Input
              type="number"
              step="0.01"
              registration={register("unit_price", {
                required: "Price is required",
                min: { value: 0.01, message: "Min 0.01" },
              })}
              error={errors.unit_price}
              min={0.01}
            />
          </FormField>
          <FormField label="Description">
            <Textarea registration={register("description")} rows={3} />
          </FormField>
          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setEditModalOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" loading={updateProduct.isPending}>
              Save Changes
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
