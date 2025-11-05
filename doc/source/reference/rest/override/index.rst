Override de configuration
#########################

Le contenu du :ref:`fichier d'override de configuration <reference-conf-override>`
peut être modifié via les handlers se trouvant en dessous du chemin ``/api/override``.

.. important::
   Pour fonctionner, l'instance doit impérativement avoir un fichier d'override,
   qui doit être créé en fournissant un nom d'instance dans :doc:`la commande de
   démarrage</reference/cli/server/server>` de Gateway.

.. attention::
   Étant donné que chaque fichier de *override* est spécifique à une
   instance particulière de Waarp Gateway, il est nécessaire que les requête faites
   sur ce *handler* soient faite directement sur l'interface d'administration de
   l'instance en question. Si Waarp Gateway est organisée en une grappe derrière
   un *load balancer*, les requêtes de modification du fichier d'override NE
   DOIVENT PAS être faites sur le *load balancer*, car il n'y a aucune garantie
   que la requête soit transmise à la bonne instance.

.. toctree::
   :maxdepth: 1

   address/index
