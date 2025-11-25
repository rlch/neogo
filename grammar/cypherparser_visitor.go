// Code generated from CypherParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package cypher // CypherParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by CypherParser.
type CypherParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by CypherParser#script.
	VisitScript(ctx *ScriptContext) interface{}

	// Visit a parse tree produced by CypherParser#query.
	VisitQuery(ctx *QueryContext) interface{}

	// Visit a parse tree produced by CypherParser#regularQuery.
	VisitRegularQuery(ctx *RegularQueryContext) interface{}

	// Visit a parse tree produced by CypherParser#singleQuery.
	VisitSingleQuery(ctx *SingleQueryContext) interface{}

	// Visit a parse tree produced by CypherParser#standaloneCall.
	VisitStandaloneCall(ctx *StandaloneCallContext) interface{}

	// Visit a parse tree produced by CypherParser#returnSt.
	VisitReturnSt(ctx *ReturnStContext) interface{}

	// Visit a parse tree produced by CypherParser#withSt.
	VisitWithSt(ctx *WithStContext) interface{}

	// Visit a parse tree produced by CypherParser#skipSt.
	VisitSkipSt(ctx *SkipStContext) interface{}

	// Visit a parse tree produced by CypherParser#limitSt.
	VisitLimitSt(ctx *LimitStContext) interface{}

	// Visit a parse tree produced by CypherParser#projectionBody.
	VisitProjectionBody(ctx *ProjectionBodyContext) interface{}

	// Visit a parse tree produced by CypherParser#projectionItems.
	VisitProjectionItems(ctx *ProjectionItemsContext) interface{}

	// Visit a parse tree produced by CypherParser#projectionItem.
	VisitProjectionItem(ctx *ProjectionItemContext) interface{}

	// Visit a parse tree produced by CypherParser#orderItem.
	VisitOrderItem(ctx *OrderItemContext) interface{}

	// Visit a parse tree produced by CypherParser#orderSt.
	VisitOrderSt(ctx *OrderStContext) interface{}

	// Visit a parse tree produced by CypherParser#singlePartQ.
	VisitSinglePartQ(ctx *SinglePartQContext) interface{}

	// Visit a parse tree produced by CypherParser#multiPartQ.
	VisitMultiPartQ(ctx *MultiPartQContext) interface{}

	// Visit a parse tree produced by CypherParser#matchSt.
	VisitMatchSt(ctx *MatchStContext) interface{}

	// Visit a parse tree produced by CypherParser#unwindSt.
	VisitUnwindSt(ctx *UnwindStContext) interface{}

	// Visit a parse tree produced by CypherParser#readingStatement.
	VisitReadingStatement(ctx *ReadingStatementContext) interface{}

	// Visit a parse tree produced by CypherParser#updatingStatement.
	VisitUpdatingStatement(ctx *UpdatingStatementContext) interface{}

	// Visit a parse tree produced by CypherParser#deleteSt.
	VisitDeleteSt(ctx *DeleteStContext) interface{}

	// Visit a parse tree produced by CypherParser#removeSt.
	VisitRemoveSt(ctx *RemoveStContext) interface{}

	// Visit a parse tree produced by CypherParser#removeItem.
	VisitRemoveItem(ctx *RemoveItemContext) interface{}

	// Visit a parse tree produced by CypherParser#queryCallSt.
	VisitQueryCallSt(ctx *QueryCallStContext) interface{}

	// Visit a parse tree produced by CypherParser#parenExpressionChain.
	VisitParenExpressionChain(ctx *ParenExpressionChainContext) interface{}

	// Visit a parse tree produced by CypherParser#yieldItems.
	VisitYieldItems(ctx *YieldItemsContext) interface{}

	// Visit a parse tree produced by CypherParser#yieldItem.
	VisitYieldItem(ctx *YieldItemContext) interface{}

	// Visit a parse tree produced by CypherParser#mergeSt.
	VisitMergeSt(ctx *MergeStContext) interface{}

	// Visit a parse tree produced by CypherParser#mergeAction.
	VisitMergeAction(ctx *MergeActionContext) interface{}

	// Visit a parse tree produced by CypherParser#setSt.
	VisitSetSt(ctx *SetStContext) interface{}

	// Visit a parse tree produced by CypherParser#setItem.
	VisitSetItem(ctx *SetItemContext) interface{}

	// Visit a parse tree produced by CypherParser#nodeLabels.
	VisitNodeLabels(ctx *NodeLabelsContext) interface{}

	// Visit a parse tree produced by CypherParser#createSt.
	VisitCreateSt(ctx *CreateStContext) interface{}

	// Visit a parse tree produced by CypherParser#patternWhere.
	VisitPatternWhere(ctx *PatternWhereContext) interface{}

	// Visit a parse tree produced by CypherParser#where.
	VisitWhere(ctx *WhereContext) interface{}

	// Visit a parse tree produced by CypherParser#pattern.
	VisitPattern(ctx *PatternContext) interface{}

	// Visit a parse tree produced by CypherParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#xorExpression.
	VisitXorExpression(ctx *XorExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#andExpression.
	VisitAndExpression(ctx *AndExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#notExpression.
	VisitNotExpression(ctx *NotExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#comparisonExpression.
	VisitComparisonExpression(ctx *ComparisonExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#comparisonSigns.
	VisitComparisonSigns(ctx *ComparisonSignsContext) interface{}

	// Visit a parse tree produced by CypherParser#addSubExpression.
	VisitAddSubExpression(ctx *AddSubExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#multDivExpression.
	VisitMultDivExpression(ctx *MultDivExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#powerExpression.
	VisitPowerExpression(ctx *PowerExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#unaryAddSubExpression.
	VisitUnaryAddSubExpression(ctx *UnaryAddSubExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#atomicExpression.
	VisitAtomicExpression(ctx *AtomicExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#listExpression.
	VisitListExpression(ctx *ListExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#stringExpression.
	VisitStringExpression(ctx *StringExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#stringExpPrefix.
	VisitStringExpPrefix(ctx *StringExpPrefixContext) interface{}

	// Visit a parse tree produced by CypherParser#nullExpression.
	VisitNullExpression(ctx *NullExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#propertyOrLabelExpression.
	VisitPropertyOrLabelExpression(ctx *PropertyOrLabelExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#propertyExpression.
	VisitPropertyExpression(ctx *PropertyExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#patternPart.
	VisitPatternPart(ctx *PatternPartContext) interface{}

	// Visit a parse tree produced by CypherParser#patternElem.
	VisitPatternElem(ctx *PatternElemContext) interface{}

	// Visit a parse tree produced by CypherParser#patternElemChain.
	VisitPatternElemChain(ctx *PatternElemChainContext) interface{}

	// Visit a parse tree produced by CypherParser#properties.
	VisitProperties(ctx *PropertiesContext) interface{}

	// Visit a parse tree produced by CypherParser#nodePattern.
	VisitNodePattern(ctx *NodePatternContext) interface{}

	// Visit a parse tree produced by CypherParser#atom.
	VisitAtom(ctx *AtomContext) interface{}

	// Visit a parse tree produced by CypherParser#lhs.
	VisitLhs(ctx *LhsContext) interface{}

	// Visit a parse tree produced by CypherParser#relationshipPattern.
	VisitRelationshipPattern(ctx *RelationshipPatternContext) interface{}

	// Visit a parse tree produced by CypherParser#relationDetail.
	VisitRelationDetail(ctx *RelationDetailContext) interface{}

	// Visit a parse tree produced by CypherParser#relationshipTypes.
	VisitRelationshipTypes(ctx *RelationshipTypesContext) interface{}

	// Visit a parse tree produced by CypherParser#unionSt.
	VisitUnionSt(ctx *UnionStContext) interface{}

	// Visit a parse tree produced by CypherParser#subqueryExist.
	VisitSubqueryExist(ctx *SubqueryExistContext) interface{}

	// Visit a parse tree produced by CypherParser#invocationName.
	VisitInvocationName(ctx *InvocationNameContext) interface{}

	// Visit a parse tree produced by CypherParser#functionInvocation.
	VisitFunctionInvocation(ctx *FunctionInvocationContext) interface{}

	// Visit a parse tree produced by CypherParser#parenthesizedExpression.
	VisitParenthesizedExpression(ctx *ParenthesizedExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#filterWith.
	VisitFilterWith(ctx *FilterWithContext) interface{}

	// Visit a parse tree produced by CypherParser#patternComprehension.
	VisitPatternComprehension(ctx *PatternComprehensionContext) interface{}

	// Visit a parse tree produced by CypherParser#relationshipsChainPattern.
	VisitRelationshipsChainPattern(ctx *RelationshipsChainPatternContext) interface{}

	// Visit a parse tree produced by CypherParser#listComprehension.
	VisitListComprehension(ctx *ListComprehensionContext) interface{}

	// Visit a parse tree produced by CypherParser#filterExpression.
	VisitFilterExpression(ctx *FilterExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#countAll.
	VisitCountAll(ctx *CountAllContext) interface{}

	// Visit a parse tree produced by CypherParser#expressionChain.
	VisitExpressionChain(ctx *ExpressionChainContext) interface{}

	// Visit a parse tree produced by CypherParser#caseExpression.
	VisitCaseExpression(ctx *CaseExpressionContext) interface{}

	// Visit a parse tree produced by CypherParser#parameter.
	VisitParameter(ctx *ParameterContext) interface{}

	// Visit a parse tree produced by CypherParser#literal.
	VisitLiteral(ctx *LiteralContext) interface{}

	// Visit a parse tree produced by CypherParser#rangeLit.
	VisitRangeLit(ctx *RangeLitContext) interface{}

	// Visit a parse tree produced by CypherParser#boolLit.
	VisitBoolLit(ctx *BoolLitContext) interface{}

	// Visit a parse tree produced by CypherParser#numLit.
	VisitNumLit(ctx *NumLitContext) interface{}

	// Visit a parse tree produced by CypherParser#stringLit.
	VisitStringLit(ctx *StringLitContext) interface{}

	// Visit a parse tree produced by CypherParser#charLit.
	VisitCharLit(ctx *CharLitContext) interface{}

	// Visit a parse tree produced by CypherParser#listLit.
	VisitListLit(ctx *ListLitContext) interface{}

	// Visit a parse tree produced by CypherParser#mapLit.
	VisitMapLit(ctx *MapLitContext) interface{}

	// Visit a parse tree produced by CypherParser#mapPair.
	VisitMapPair(ctx *MapPairContext) interface{}

	// Visit a parse tree produced by CypherParser#name.
	VisitName(ctx *NameContext) interface{}

	// Visit a parse tree produced by CypherParser#symbol.
	VisitSymbol(ctx *SymbolContext) interface{}

	// Visit a parse tree produced by CypherParser#reservedWord.
	VisitReservedWord(ctx *ReservedWordContext) interface{}
}
