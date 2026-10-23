.. _user-webui-overrides:

############################
Surcharges de configuration
############################

Section *Administration*, adresse ``/webui/overrides``. La consultation requiert
le droit de lecture sur l'administration ; l'ajout et la modification d'une
règle, le droit d'écriture ; sa suppression, le droit de suppression. Ce sont
les droits de l'API REST des surcharges.

Dans une grappe, toutes les instances de la Gateway partagent la même
configuration : un serveur local y est déclaré une fois, avec l'adresse publique
du répartiteur de charge, qu'aucun nœud ne peut écouter lui-même. Une
**surcharge** est un réglage propre à un nœud, qui l'emporte sur la
configuration commune (voir :ref:`la référence des surcharges
<reference-conf-override>`).

Aujourd'hui, la Gateway connaît une sorte de surcharge : l'**indirection
d'adresse**, que l'écran présente comme une adresse remplacée. Sur ce nœud,
chaque fois que la configuration commune donne l'adresse configurée, la Gateway
emploie l'adresse du nœud :

- pour écouter : l'administration (API REST et interface web) et les serveurs
  locaux ;
- pour se connecter aux partenaires.

Une règle porte un hôte seul (``lb.example.org`` remplacé par ``10.0.0.5`` :
l'hôte est remplacé, le port conservé) ou un hôte et son port
(``lb.example.org:2222`` remplacé par ``10.0.0.5:22222``) ; les deux adresses
ont la même forme. Quand une adresse correspond à la fois à une règle avec port
et à une règle sur l'hôte seul, la règle avec port l'emporte.

Les règles vivent dans un fichier propre au nœud, ``<nom d'instance>.ini``, à
côté du fichier de configuration, et non en base. Elles n'existent que si la
Gateway démarre avec un nom d'instance (voir :doc:`la commande de démarrage
</reference/cli/server/server>`) ; ce fichier doit alors exister, sinon la Gateway refuse
de démarrer.


La liste
========

L'en-tête de la page donne le nom du nœud. Dans une grappe, la page montre les
règles du nœud qui la sert, c'est-à-dire celui vers lequel le répartiteur de
charge a dirigé le navigateur.

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Adresse configurée
     - L'adresse telle qu'elle figure dans la configuration commune.
   * - Adresse sur ce nœud
     - L'adresse que le nœud emploie à sa place.
   * - Portée
     - *Hôte et port*, ou *Hôte seul, port conservé*.
   * - Concerne
     - Ce que la règle remplace dans la configuration commune : un serveur
       local ou un partenaire (son nom, qui ouvre sa fiche), un nombre de
       serveurs ou de partenaires, l'administration ; *Rien pour l'instant*
       sinon.
   * - Actions
     - Le menu « ⋯ » de la ligne : *Modifier la règle* (droit d'écriture),
       *Supprimer la règle* (droit de suppression).

La colonne *Adresse configurée* se trie d'un clic sur son en-tête.

Une règle qui concerne l'adresse de l'administration porte une mise en garde :
au prochain démarrage de la Gateway, l'API et l'interface web écouteront sur
l'adresse du nœud. Vérifiez qu'elle reste joignable avant de redémarrer.

Sur une instance démarrée sans nom d'instance, la page l'explique et ne propose
aucune action : elle ne fait pas partie d'une grappe.


Quand une règle s'applique
==========================

Une règle ajoutée, modifiée ou supprimée est écrite aussitôt dans le fichier du
nœud. Elle s'applique :

- aux serveurs locaux concernés, à leur prochain démarrage : un serveur en
  marche garde son adresse jusqu'à ce qu'on le redémarre (écran
  :doc:`serveurs-locaux`) ;
- aux connexions vers les partenaires, aussitôt ;
- à l'administration, au prochain démarrage de la Gateway.

L'adresse enregistrée dans la configuration commune ne change pas.


Ajouter, modifier ou supprimer une règle
========================================

*Ajouter une adresse* ouvre le formulaire : l'adresse configurée, telle qu'elle
figure dans la configuration commune, et l'adresse sur ce nœud, de la même
forme. Chaque adresse est un nom d'hôte ou une adresse IP, suivi au besoin de
``:port`` (port de 1 à 65535) ; une adresse IPv6 s'écrit entre crochets
(``[2001:db8::1]:443``). Le formulaire refuse une adresse déjà remplacée et
renvoie à la modification de sa règle.

*Modifier la règle* ne change que l'adresse sur ce nœud : l'adresse configurée
est la clé de la règle. Pour la changer, supprimez la règle et ajoutez-en une.

*Supprimer la règle* se confirme sur une page qui la rappelle avec ce qu'elle
concerne. Après la suppression, le nœud emploie de nouveau l'adresse configurée
telle quelle, selon les mêmes délais.

Le fichier du nœud est réécrit à chaque modification : un commentaire qu'on y
aurait écrit à la main disparaît.


Et par l'API
============

Les mêmes règles se gèrent par l':doc:`API REST </reference/rest/override/index>`
(``/api/override/addresses``) et par la ligne de commande
(``waarp-gateway override address``). Une règle écrite par l'un s'affiche et se
modifie par les autres.

L'ancienne page *Surcharges de configuration*, qui renvoyait à la ligne de
commande, mène désormais à cet écran.
