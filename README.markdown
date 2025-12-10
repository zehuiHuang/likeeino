eino官网地址:https://www.cloudwego.io/zh/docs/eino/

# agent: reactAgent、workflowAgents、multiAgent

# reactAgent:
模型和工具结合,是一个智能决策的大脑,常用场景知识问答、IT运维(可根据线索一步步定位问题)

# workflowAgents: 
定制化流水线,按照顺序严格执行,常用场景比如CICD、数据迁移等

# multiAgent:
一种是集中式协调(Supervisor),中心化设计,主agent管理一批子agent,主agent可以根据情况来进行动态调整和任务分配给子agent;
另外一种是结构化问题解决( Plan-Execute),


^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^各组件定义和使用

---------------------------------------------------------------Graph 编排
一、Graph 编排
1、使用时
将各个组件先添加到graph,然后通过AddEdge将各个组件串联起来,首尾分别是:compose.START和compose.END,其他组件(名称)在中间,
并且按照头尾相连.
各组件包括:AddChatTemplateNode、AddChatModelNode、AddToolsNode、
AddLambdaNode、AddRetrieverNode、AddEdge、AddBranch、AddDocumentTransformerNode、AddEmbeddingNode、AddIndexerNode等

2、全局状态

---------------------------------------------------------------Graph 编排





---------------------------------------------------------------人机协同
//代码演示路径adk/human-in-the-loop/
二、人机协同
# 背景说明:
在程序执行的过程中,需要人的参与(比如确认信息是否正确、内容是否需要优化),
通过外部的参数接入来影响程序后续的执行(中断、执行、调整重新执行、循环执行等)
# 场景案例
1、确认信息,根据信息人为的中断或继续
2、修改信息,参考工具返回的参数信息,对参数信息进行调整后在执行
3、给agent返回的数据人为的提出复合自身要求的描述,agent会以此为条件执行并给出结果,并无限循环,直至人为要求的进行最终答案的输出
4、通过多轮对话,识别用户对话,制定出更符合预期的答案(常适用于咨询、问题收集等场景)

---------------------------------------------------------------人机协同