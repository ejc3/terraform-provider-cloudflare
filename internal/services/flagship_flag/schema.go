// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package flagship_flag

import (
	"context"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/schemata"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigValidators = (*FlagshipFlagResource)(nil)

func ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: schemata.Description{
			Scopes: []string{
				"Flagship Read",
				"Flagship Write",
			},
		}.String(),
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Description:   "Cloudflare account ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"app_id": schema.StringAttribute{
				Description:   "App identifier.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"flag_key": schema.StringAttribute{
				Description:   "Flag key (slug).",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"default_variation": schema.StringAttribute{
				Description: "Variation the API serves when the flag is off, or when it's on but no rule matches the context. Must be a key in `variations`.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "When false, the flag bypasses all rules and always serves `default_variation`.",
				Required:    true,
			},
			"key": schema.StringAttribute{
				Description: "Unique identifier for the flag within an app. Used in all evaluation and SDK calls.",
				Required:    true,
			},
			"variations": schema.MapAttribute{
				Description: "Map of variation name to value. All values share the same type (boolean, string, number, or JSON object/array), and each serialized value stays within 10KB.",
				Required:    true,
				ElementType: types.StringType,
			},
			"rules": schema.ListNestedAttribute{
				Description: "Targeting rules evaluated in ascending `priority`; the first matching rule wins. An empty array means the flag always serves `default_variation`.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"conditions": schema.ListNestedAttribute{
							Description: "Conditions the context must satisfy for this rule to match. An empty array matches all contexts.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"attribute": schema.StringAttribute{
										Optional: true,
									},
									"operator": schema.StringAttribute{
										Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
										Optional:    true,
										Validators: []validator.String{
											stringvalidator.OneOfCaseInsensitive(
												"equals",
												"not_equals",
												"greater_than",
												"less_than",
												"greater_than_or_equals",
												"less_than_or_equals",
												"contains",
												"starts_with",
												"ends_with",
												"in",
												"not_in",
											),
										},
									},
									"value": schema.StringAttribute{
										Optional: true,
									},
									"clauses": schema.ListNestedAttribute{
										Optional: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"attribute": schema.StringAttribute{
													Optional: true,
												},
												"operator": schema.StringAttribute{
													Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
													Optional:    true,
													Validators: []validator.String{
														stringvalidator.OneOfCaseInsensitive(
															"equals",
															"not_equals",
															"greater_than",
															"less_than",
															"greater_than_or_equals",
															"less_than_or_equals",
															"contains",
															"starts_with",
															"ends_with",
															"in",
															"not_in",
														),
													},
												},
												"value": schema.StringAttribute{
													Optional: true,
												},
												"clauses": schema.ListNestedAttribute{
													Optional: true,
													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"attribute": schema.StringAttribute{
																Optional: true,
															},
															"operator": schema.StringAttribute{
																Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
																Optional:    true,
																Validators: []validator.String{
																	stringvalidator.OneOfCaseInsensitive(
																		"equals",
																		"not_equals",
																		"greater_than",
																		"less_than",
																		"greater_than_or_equals",
																		"less_than_or_equals",
																		"contains",
																		"starts_with",
																		"ends_with",
																		"in",
																		"not_in",
																	),
																},
															},
															"value": schema.StringAttribute{
																Optional: true,
															},
															"clauses": schema.ListNestedAttribute{
																Optional: true,
																NestedObject: schema.NestedAttributeObject{
																	Attributes: map[string]schema.Attribute{
																		"attribute": schema.StringAttribute{
																			Optional: true,
																		},
																		"operator": schema.StringAttribute{
																			Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
																			Optional:    true,
																			Validators: []validator.String{
																				stringvalidator.OneOfCaseInsensitive(
																					"equals",
																					"not_equals",
																					"greater_than",
																					"less_than",
																					"greater_than_or_equals",
																					"less_than_or_equals",
																					"contains",
																					"starts_with",
																					"ends_with",
																					"in",
																					"not_in",
																				),
																			},
																		},
																		"value": schema.StringAttribute{
																			Optional: true,
																		},
																		"clauses": schema.ListNestedAttribute{
																			Optional: true,
																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"attribute": schema.StringAttribute{
																						Optional: true,
																					},
																					"operator": schema.StringAttribute{
																						Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
																						Optional:    true,
																						Validators: []validator.String{
																							stringvalidator.OneOfCaseInsensitive(
																								"equals",
																								"not_equals",
																								"greater_than",
																								"less_than",
																								"greater_than_or_equals",
																								"less_than_or_equals",
																								"contains",
																								"starts_with",
																								"ends_with",
																								"in",
																								"not_in",
																							),
																						},
																					},
																					"value": schema.StringAttribute{
																						Optional: true,
																					},
																					"clauses": schema.ListNestedAttribute{
																						Optional: true,
																						NestedObject: schema.NestedAttributeObject{
																							Attributes: map[string]schema.Attribute{
																								"attribute": schema.StringAttribute{
																									Optional: true,
																								},
																								"operator": schema.StringAttribute{
																									Description: `Available values: "equals", "not_equals", "greater_than", "less_than", "greater_than_or_equals", "less_than_or_equals", "contains", "starts_with", "ends_with", "in", "not_in".`,
																									Optional:    true,
																									Validators: []validator.String{
																										stringvalidator.OneOfCaseInsensitive(
																											"equals",
																											"not_equals",
																											"greater_than",
																											"less_than",
																											"greater_than_or_equals",
																											"less_than_or_equals",
																											"contains",
																											"starts_with",
																											"ends_with",
																											"in",
																											"not_in",
																										),
																									},
																								},
																								"value": schema.StringAttribute{
																									Optional: true,
																								},
																								"clauses": schema.ListAttribute{
																									Optional:    true,
																									ElementType: types.StringType,
																								},
																								"logical_operator": schema.StringAttribute{
																									Description: `Available values: "AND", "OR".`,
																									Optional:    true,
																									Validators: []validator.String{
																										stringvalidator.OneOfCaseInsensitive("AND", "OR"),
																									},
																								},
																							},
																						},
																					},
																					"logical_operator": schema.StringAttribute{
																						Description: `Available values: "AND", "OR".`,
																						Optional:    true,
																						Validators: []validator.String{
																							stringvalidator.OneOfCaseInsensitive("AND", "OR"),
																						},
																					},
																				},
																			},
																		},
																		"logical_operator": schema.StringAttribute{
																			Description: `Available values: "AND", "OR".`,
																			Optional:    true,
																			Validators: []validator.String{
																				stringvalidator.OneOfCaseInsensitive("AND", "OR"),
																			},
																		},
																	},
																},
															},
															"logical_operator": schema.StringAttribute{
																Description: `Available values: "AND", "OR".`,
																Optional:    true,
																Validators: []validator.String{
																	stringvalidator.OneOfCaseInsensitive("AND", "OR"),
																},
															},
														},
													},
												},
												"logical_operator": schema.StringAttribute{
													Description: `Available values: "AND", "OR".`,
													Optional:    true,
													Validators: []validator.String{
														stringvalidator.OneOfCaseInsensitive("AND", "OR"),
													},
												},
											},
										},
									},
									"logical_operator": schema.StringAttribute{
										Description: `Available values: "AND", "OR".`,
										Optional:    true,
										Validators: []validator.String{
											stringvalidator.OneOfCaseInsensitive("AND", "OR"),
										},
									},
								},
							},
						},
						"priority": schema.Int64Attribute{
							Description: "Evaluation order: the API evaluates rules with lower numbers first. Must be unique across the flag's rules.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"serve_variation": schema.StringAttribute{
							Description: "Variation the API serves when this rule matches. Must be a key in `variations`.",
							Required:    true,
						},
						"rollout": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"percentage": schema.Float64Attribute{
									Description: "Percentage of matching traffic (0–100) served this variation. For multi-way splits, use cumulative upper bounds across rules (e.g. 30, 70, 100).",
									Required:    true,
									Validators: []validator.Float64{
										float64validator.Between(0, 100),
									},
								},
								"attribute": schema.StringAttribute{
									Description: "Context attribute used for sticky bucketing. Defaults to `targetingKey`. If absent at evaluation time, bucketing is random per request.",
									Optional:    true,
								},
							},
						},
					},
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"type": schema.StringAttribute{
				Description: "Value type of the flag's variations. The API infers this from the variation values on write, so you can omit it in requests.\nAvailable values: \"boolean\", \"string\", \"number\", \"json\".",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"boolean",
						"string",
						"number",
						"json",
					),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_by": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *FlagshipFlagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}

func (r *FlagshipFlagResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}
