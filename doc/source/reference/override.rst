.. _reference-conf-override:

Override de configuration
#########################

Chaque instance de Gateway a sa propre configuration (définie dans son
:any:`fichier de configuration <configuration>`). Cependant, lorsque
Gateway fonctionne en grappe avec plusieurs instances, cette configuration est
commune à toutes les instances de la grappe (puisque la grappe est considérée
comme une unique entité). Or, il est parfois nécessaire d'avoir des
paramètres spécifiques pour chaque instance de la grappe.

Pour palier à ce problème, il est possible d'utiliser un fichier de *override*.
Au démarrage de Waarp Gateway, une des options de la commande de démarrage permet
de renseigner un nom d'instance (voir :any:`la documentation de la commande <cli/server/server>`).
Ce nom d'instance permet de différentier une instance de Gateway des autres
instances de la grappe, et doit donc être unique au sein d'une même grappe.
Ce nom d'instance est **obligatoire** dans le cadre du fonctionnement en grappe.

Au démarrage, Waarp Gateway cherchera un fichier .ini portant son nom d'instance
dans le dossier du fichier de configuration principal. Si ce fichier existe,
tous les paramètres définis dedans prendront précédence sur ceux définis dans le
fichier de configuration principal ou dans la base de données.

Pour l'instant, ce fichier ne permet que de définir des indirections d'adresse.

.. module:: <instance>.ini
   :synopsis: fichier de *override* de configuration du démon waarp-gatewayd

.. _reference-address-indirection:

Section ``[Address Indirection]``
=================================

La section ``[Address Indirection]`` contient une liste d'indirection d'adresses.
Une indirection d'adresse permet de remplacer dynamiquement certaines adresses
(IP ou DNS) définies dans la configuration par d'autres.

Par exemple, dans une grappe d'instances Gateway, les serveurs locaux sont définis au
niveau de la grappe entière (puisque les nœuds de la grappe sont tous des clones
les uns des autres). Dans cette situation, l'adresse renseignée lors de la définition
d'un serveur local est l'adresse du proxy/*load balancer*. Les instances de la
grappe ne peuvent, elles, pas écouter sur cette adresse puisqu'elles n'y ont pas
accès. C'est la que l'indirection d'adresse entre en jeu, puisqu'elle va permettre
pour chaque instance de la grappe de dynamiquement remplacer cette adresse publique
du *load balancer* par l'adresse locale de l'interface réseau sur laquelle l'instance
doit écouter, tout en conservant le reste de la configuration commun à toutes les
instances.

.. confval:: IndirectAddress

   Définit une indirection d'adresse sous la forme :

   .. code-block:: ini

      IndirectAddress = <adresse à remplacer> -> <adresse de remplacement>

   Les adresse avec et sans port sont acceptées.  Ce paramètre peut être répété
   autant de fois qu'il y a d'indirections. Exemple :

   .. code-block:: ini

      IndirectAddress = localhost -> 127.0.0.1
      IndirectAddress = example.com -> 8.8.8.8:80
