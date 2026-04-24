"use client";

import { useRouter } from "next/navigation";
import { useForm, useFieldArray } from "react-hook-form";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import Button from "@/components/ui/Button";
import { FormField, Input } from "@/components/ui/FormField";
import { Card } from "@/components/ui/Card";
import { useCreateOrder, type CreateOrderInput } from "@/lib/use-orders";
import { toastSuccess, toastError } from "@/lib/toast";

interface FormValues {
  customer_name: string;
  items: { product_id: string; name: string; quantity: string; unit_price: string }[];
}

export default function NewOrderPage() {
  const router = useRouter();
  const createOrder = useCreateOrder();

  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors },
  } = useForm<FormValues>({
    defaultValues: {
      customer_name: "",
      items: [{ product_id: "", name: "", quantity: "1", unit_price: "" }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "items",
  });

  const watchedItems = watch("items");

  const totalAmount = watchedItems.reduce((sum, item) => {
    const qty = parseFloat(item.quantity) || 0;
    const price = parseFloat(item.unit_price) || 0;
    return sum + qty * price;
  }, 0);

  const onSubmit = (data: FormValues) => {
    const input: CreateOrderInput = {
      customer_name: data.customer_name,
      items: data.items.map((item) => ({
        product_id: item.product_id || undefined as unknown as string,
        name: item.name,
        quantity: parseInt(item.quantity, 10),
        unit_price: parseFloat(item.unit_price),
      })),
    };

    createOrder.mutate(input, {
      onSuccess: (res) => {
        toastSuccess("Order created successfully");
        router.push(`/orders/${res.data.id}`);
      },
      onError: toastError,
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/orders")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <h1 className="text-2xl font-bold text-gray-900">Create Order</h1>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {}
        <Card>
          <FormField
            label="Customer Name"
            required
            error={errors.customer_name}
          >
            <Input
              registration={register("customer_name", {
                required: "Customer name is required",
              })}
              error={errors.customer_name}
              placeholder="Enter customer name"
            />
          </FormField>
        </Card>

        {}
        <Card>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-700">Line Items</h2>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() =>
                append({ product_id: "", name: "", quantity: "1", unit_price: "" })
              }
            >
              <Plus className="h-3.5 w-3.5" />
              Add Item
            </Button>
          </div>

          <div className="space-y-4">
            {fields.map((field, index) => (
              <div
                key={field.id}
                className="grid grid-cols-12 gap-3 items-end rounded-md border border-gray-100 bg-gray-50 p-3"
              >
                <div className="col-span-4">
                  <FormField
                    label="Product Name"
                    required
                    error={errors.items?.[index]?.name}
                  >
                    <Input
                      registration={register(`items.${index}.name`, {
                        required: "Required",
                      })}
                      error={errors.items?.[index]?.name}
                      placeholder="Product name"
                    />
                  </FormField>
                </div>
                <div className="col-span-3">
                  <FormField label="Product ID">
                    <Input
                      registration={register(`items.${index}.product_id`)}
                      placeholder="Optional"
                    />
                  </FormField>
                </div>
                <div className="col-span-2">
                  <FormField
                    label="Qty"
                    required
                    error={errors.items?.[index]?.quantity}
                  >
                    <Input
                      type="number"
                      registration={register(`items.${index}.quantity`, {
                        required: "Required",
                        min: { value: 1, message: "Min 1" },
                      })}
                      error={errors.items?.[index]?.quantity}
                      min={1}
                    />
                  </FormField>
                </div>
                <div className="col-span-2">
                  <FormField
                    label="Unit Price"
                    required
                    error={errors.items?.[index]?.unit_price}
                  >
                    <Input
                      type="number"
                      step="0.01"
                      registration={register(`items.${index}.unit_price`, {
                        required: "Required",
                        min: { value: 0.01, message: "Min 0.01" },
                      })}
                      error={errors.items?.[index]?.unit_price}
                      min={0.01}
                    />
                  </FormField>
                </div>
                <div className="col-span-1 flex justify-center pb-0.5">
                  {fields.length > 1 && (
                    <button
                      type="button"
                      onClick={() => remove(index)}
                      className="rounded-md p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>

          {}
          <div className="mt-4 flex justify-end border-t pt-3">
            <div className="text-right">
              <p className="text-sm text-gray-500">Estimated Total</p>
              <p className="text-xl font-bold text-gray-900">
                ${totalAmount.toFixed(2)}
              </p>
            </div>
          </div>
        </Card>

        {}
        <div className="flex justify-end gap-3">
          <Button
            type="button"
            variant="secondary"
            onClick={() => router.push("/orders")}
          >
            Cancel
          </Button>
          <Button type="submit" loading={createOrder.isPending}>
            Create Order
          </Button>
        </div>
      </form>
    </div>
  );
}
