##################
Gestion des règles
##################

Le point d'accès pour gérer les règles de transfert est ``/api/rules``.

L'utilisation d'une règle peut être restreinte via le point d'accès REST
``/api/rules/<rule_id>/access``.

Les traitements rattachés à une règle peuvent être consultés et modifiés à
l'aide du point d'accès ``/api/rules/<rule_id>/tasks``.

.. toctree::
   :maxdepth: 2

   create
   list
   consult
   delete
   access/index
   tasks/index