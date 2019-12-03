#############################
Gestion de l'accès aux règles
#############################

Le point d'accès pour gérer l'accès aux règles de transfert est
``/api/rules/<rule_id>/access``. L'accès à une règle peut être accordé ou révoqué
en envoyant les informations nécessaires en format JSON dans une requête REST
avec la méthode appropriée (`POST` ou `DELETE`).

Une règle n'ayant aucun accès définit est considérée comme étant universellement
accessible.

.. toctree::
   :maxdepth: 1

   create
   list
   delete