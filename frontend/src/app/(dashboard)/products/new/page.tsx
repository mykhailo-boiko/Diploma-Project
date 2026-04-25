"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { ArrowLeft } from "lucide-react";
import Button from "@/components/ui/Button";
import { FormField, Input, Select, Textarea } from "@/components/ui/FormField";
import { Card } from "@/components/ui/Card";
import { useCreateProduct, type CreateProductInput } from "@/lib/use-products";
import { toastSuccess, toastError } from "@/lib/toast";

const CATEGORY_OPTIONS = [
  { value: "", label: "Select category..." },
  { value: "Electronics", label: "Electronics" },
  { value: "Furniture", label: "Furniture" },
  { value: "Clothing", label: "Clothing" },
  { value: "Food & Beverage", label: "Food & Beverage" },
  { value: "Tools", label: "Tools" },
];

interface FormValues {
  sku: string;
  name: string;
  description: string;
  category: string;
  unit_price: string;
}

export default function NewProductPage() {
  const router = useRouter();
  const createProduct = useCreateProduct();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    defaultValues: {
      sku: "",
      name: "",
      description: "",
      category: "",
      unit_price: "",
    },
  });

  const onSubmit = (data: FormValues) => {
    const input: CreateProductInput = {
      sku: data.sku,
      name: data.name,
      description: data.description || undefined,
      category: data.category || undefined,
      unit_price: parseFloat(data.unit_price),
    };

    createProduct.mutate(input, {
      onSuccess: (res) => {
        toastSuccess("Product created successfully");
        router.push(`/products/${res.data.id}`);
      },
      onError: toastError,
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/products")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <h1 className="text-2xl font-bold text-gray-900">Create Product</h1>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        <Card>
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="SKU" required error={errors.sku}>
              <Input
                registration={register("sku", { required: "SKU is required" })}
                error={errors.sku}
                placeholder="e.g. ELEC-001"
              />
            </FormField>
            <FormField label="Name" required error={errors.name}>
              <Input
                registration={register("name", { required: "Name is required" })}
                error={errors.name}
                placeholder="Product name"
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
                placeholder="0.00"
                min={0.01}
              />
            </FormField>
          </div>
          <div className="mt-4">
            <FormField label="Description">
              <Textarea
                registration={register("description")}
                placeholder="Product description (optional)"
                rows={3}
              />
            </FormField>
          </div>
        </Card>

        <div className="flex justify-end gap-3">
          <Button
            type="button"
            variant="secondary"
            onClick={() => router.push("/products")}
          >
            Cancel
          </Button>
          <Button type="submit" loading={createProduct.isPending}>
            Create Product
          </Button>
        </div>
      </form>
    </div>
  );
}
