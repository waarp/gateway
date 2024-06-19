#######################################
Gestion des *override* de configuration
#######################################

Le contenu du :any:`fichier d'override de configuration <../../../override>`
peut être modifié via la commande ``override`` et ses sous-commandes.

.. warning::
   Étant donné que chaque fichier de *override* est spécifique à une instance
   particulière de Waarp Gateway, il est nécessaire que le client
   ``waarp-gateway`` ait un accès réseau direct à l'instance à modifier. Si
   l'instance à modifier se trouve derrière un *load balancer*, il n'y a aucune
   garantie que les commandes soient transmises à la bonne instance.

.. toctree::
   :maxdepth: 2

   address/index

