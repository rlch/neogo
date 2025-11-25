// Code generated from CypherParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package cypher // CypherParser
import "github.com/antlr4-go/antlr/v4"

type BaseCypherParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseCypherParserVisitor) VisitScript(ctx *ScriptContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitQuery(ctx *QueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRegularQuery(ctx *RegularQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSingleQuery(ctx *SingleQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitStandaloneCall(ctx *StandaloneCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitReturnSt(ctx *ReturnStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitWithSt(ctx *WithStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSkipSt(ctx *SkipStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitLimitSt(ctx *LimitStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitProjectionBody(ctx *ProjectionBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitProjectionItems(ctx *ProjectionItemsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitProjectionItem(ctx *ProjectionItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitOrderItem(ctx *OrderItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitOrderSt(ctx *OrderStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSinglePartQ(ctx *SinglePartQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMultiPartQ(ctx *MultiPartQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMatchSt(ctx *MatchStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitUnwindSt(ctx *UnwindStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitReadingStatement(ctx *ReadingStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitUpdatingStatement(ctx *UpdatingStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitDeleteSt(ctx *DeleteStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRemoveSt(ctx *RemoveStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRemoveItem(ctx *RemoveItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitQueryCallSt(ctx *QueryCallStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitParenExpressionChain(ctx *ParenExpressionChainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitYieldItems(ctx *YieldItemsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitYieldItem(ctx *YieldItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMergeSt(ctx *MergeStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMergeAction(ctx *MergeActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSetSt(ctx *SetStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSetItem(ctx *SetItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitNodeLabels(ctx *NodeLabelsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitCreateSt(ctx *CreateStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPatternWhere(ctx *PatternWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitWhere(ctx *WhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPattern(ctx *PatternContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitXorExpression(ctx *XorExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitAndExpression(ctx *AndExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitNotExpression(ctx *NotExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitComparisonExpression(ctx *ComparisonExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitComparisonSigns(ctx *ComparisonSignsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitAddSubExpression(ctx *AddSubExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMultDivExpression(ctx *MultDivExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPowerExpression(ctx *PowerExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitUnaryAddSubExpression(ctx *UnaryAddSubExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitAtomicExpression(ctx *AtomicExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitListExpression(ctx *ListExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitStringExpression(ctx *StringExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitStringExpPrefix(ctx *StringExpPrefixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitNullExpression(ctx *NullExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPropertyOrLabelExpression(ctx *PropertyOrLabelExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPropertyExpression(ctx *PropertyExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPatternPart(ctx *PatternPartContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPatternElem(ctx *PatternElemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPatternElemChain(ctx *PatternElemChainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitProperties(ctx *PropertiesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitNodePattern(ctx *NodePatternContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitAtom(ctx *AtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitLhs(ctx *LhsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRelationshipPattern(ctx *RelationshipPatternContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRelationDetail(ctx *RelationDetailContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRelationshipTypes(ctx *RelationshipTypesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitUnionSt(ctx *UnionStContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSubqueryExist(ctx *SubqueryExistContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitInvocationName(ctx *InvocationNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitFunctionInvocation(ctx *FunctionInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitParenthesizedExpression(ctx *ParenthesizedExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitFilterWith(ctx *FilterWithContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitPatternComprehension(ctx *PatternComprehensionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRelationshipsChainPattern(ctx *RelationshipsChainPatternContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitListComprehension(ctx *ListComprehensionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitFilterExpression(ctx *FilterExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitCountAll(ctx *CountAllContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitExpressionChain(ctx *ExpressionChainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitCaseExpression(ctx *CaseExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitParameter(ctx *ParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitLiteral(ctx *LiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitRangeLit(ctx *RangeLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitBoolLit(ctx *BoolLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitNumLit(ctx *NumLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitStringLit(ctx *StringLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitCharLit(ctx *CharLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitListLit(ctx *ListLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMapLit(ctx *MapLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitMapPair(ctx *MapPairContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitName(ctx *NameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitSymbol(ctx *SymbolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseCypherParserVisitor) VisitReservedWord(ctx *ReservedWordContext) interface{} {
	return v.VisitChildren(ctx)
}
