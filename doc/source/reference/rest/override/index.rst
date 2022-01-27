Override de configuration
#########################

Le contenu du :ref:`fichier d'override de configuration <reference-conf-override>`
peut être modifié via les handlers se trouvant en dessous du chemin ``/api/override``.

.. attention:: Étant donné que chaque fichier de *override* est spécifique à une
   instance particulière de *gateway*, il est nécessaire que les requête faites
   sur ce *handler* soient faite directement sur l'interface d'administration de
   l'instance en question. Si la *gateway* est organisée en une grappe derrière
   un *load balancer*, les requêtes de modification du fichier d'override NE
   DOIVENT PAS être faites sur le *load balancer*, car il n'y a aucune garantie
   que la requête soit transmise à la bonne instance.

.. toctree::
   :maxdepth: 1

   address/index