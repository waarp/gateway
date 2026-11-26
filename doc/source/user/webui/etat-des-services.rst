.. _user-webui-services:

##################
État des services
##################

Section *Supervision*, adresse ``/webui/services``. Aucun droit particulier
n'est requis : tout utilisateur authentifié peut consulter cet écran.

L'écran montre l'état d'exécution des composants de la Gateway, en trois
groupes :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Groupe
     - Contenu
   * - Services système
     - Les cinq composants internes, toujours présents : administration (API
       REST et interface web), statistiques internes, contrôleur des
       transferts, base de données, agent SNMP. Le nom technique du composant
       (``Admin``, ``Controller``…) est affiché sous son libellé.
   * - Serveurs locaux
     - Un service par :ref:`serveur local <user-webui-servers>` défini sur la
       Gateway.
   * - Clients locaux
     - Un service par client local défini sur la Gateway.

Trois tuiles, en tête de page, résument chaque groupe : nombre de services en
ligne sur le total, et ce qui ne va pas le cas échéant. Un seul tableau est
affiché à la fois, celui de la tuile sélectionnée (la première à l'ouverture) ;
cliquer une autre tuile remplace le tableau, et l'adresse de la page garde le
groupe choisi. Les colonnes du nom et de l'état se trient d'un clic sur leur
en-tête (par défaut, le nom croissant ; l'état se trie du plus sain au plus
dégradé). Au-delà de 10 lignes, le tableau est paginé : le pied de liste
indique les lignes affichées et permet de changer de page.


États
=====

Chaque service porte un état, signalé par une forme, une couleur et un
libellé, lisible sans la couleur :

.. list-table::
   :header-rows: 1
   :widths: 20 15 65

   * - État
     - Forme
     - Signification
   * - En ligne
     - Rond vert
     - Le service fonctionne.
   * - Hors ligne
     - Anneau gris
     - Le service est arrêté volontairement : désactivé, ou arrêté par un
       administrateur depuis les écrans *Serveurs locaux* ou *Clients
       locaux*.
   * - Erreur
     - Losange rouge
     - Le service ne fonctionne pas. La colonne *Raison* donne le message
       d'erreur tel que le service l'a produit (par exemple un port déjà
       utilisé).
   * - Inconnu
     - Triangle orange
     - Le service a renvoyé un état hors des trois précédents.

La colonne *Raison* n'est renseignée qu'en cas d'erreur ; un tiret indique
l'absence de raison.

Cet écran ne porte aucune action : démarrer ou arrêter un serveur ou un
client se fait depuis les écrans *Serveurs locaux* et *Clients locaux*.


Actualisation
=============

Les états sont actualisés automatiquement toutes les 10 secondes, sans
recharger la page ; l'heure de la dernière actualisation est affichée à côté
du bouton *Actualiser*, qui force une actualisation immédiate. Le groupe, le
tri et la page choisis sont conservés d'une actualisation à l'autre.

Quand une actualisation échoue (Gateway injoignable, session expirée), un
message persistant l'indique en tête des tableaux, avec l'heure de la
dernière lecture réussie ; les états affichés datent de cette lecture. Les
tentatives se poursuivent toutes les 10 secondes ; le bouton *Réessayer*
en lance une immédiatement. Si la session a expiré, le bouton *Actualiser*
ramène à la page de connexion.

Sans JavaScript, la page est complète mais ne s'actualise pas d'elle-même :
*Actualiser* recharge alors la page.


Et par l'API
============

Les mêmes informations sont disponibles par l'API REST, sur le point
:http:get:`/api/about` (voir :ref:`reference-rest-api`).
