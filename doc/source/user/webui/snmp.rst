.. _user-webui-snmp:

####
SNMP
####

Section *Administration*, adresse ``/webui/snmp``. La consultation requiert le
droit de lecture sur l'administration ; l'ajout et la modification d'un
moniteur, la configuration de l'agent, sa désactivation et la notification de
test, le droit d'écriture ; la suppression d'un moniteur, le droit de
suppression. Ce sont les droits de l'API REST de SNMP.

SNMP est le protocole des outils de supervision. La Gateway s'en sert de deux
façons :

- **Les notifications** : en cas d'erreur de transfert ou de défaillance d'un
  service, elle envoie une notification (*trap*) à chaque :term:`moniteur SNMP`,
  c'est-à-dire à l'outil de supervision qui la reçoit.
- **L'agent** : s'il est configuré, la Gateway répond aux requêtes de lecture
  d'un outil de supervision : version, durée de fonctionnement, mémoire,
  connexions, nombre de transferts par état et par code d'erreur, dernier
  transfert. Les objets lus sont décrits par la :ref:`MIB de la Gateway
  <reference-snmp-mib>`.

Deux niveaux de sécurité coexistent. En SNMPv2c, une **communauté** sert de mot
de passe ; elle circule en clair. SNMPv3 ajoute un utilisateur, une
authentification et un chiffrement, chacun avec sa phrase secrète. Les phrases
secrètes et les communautés ne sont jamais réaffichées par l'interface : la
page dit seulement si elles sont définies.


La page SNMP
============

La page montre les deux sens de SNMP, chacun dans un bloc titré par ce qu'il
fait et introduit par une phrase : *Interrogation par la supervision* (l'agent,
voir plus bas), puis *Alertes envoyées par la passerelle* (les moniteurs).

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Moniteur
     - Le nom du moniteur ; il ouvre sa fiche.
   * - Version
     - *SNMPv2c* ou *SNMPv3*.
   * - Adresse
     - L'adresse UDP de l'outil de supervision, sous la forme ``hôte:port``.
   * - Sécurité
     - *Communauté* en SNMPv2c ; en SNMPv3, le niveau : *Sans
       authentification*, *Authentification* ou *Authentification et
       chiffrement*.
   * - Actions
     - Le menu « ⋯ » de la ligne : *Voir la fiche* ; *Modifier le moniteur*
       (droit d'écriture) ; *Supprimer le moniteur* (droit de suppression).

Les colonnes *Moniteur* et *Adresse* se trient d'un clic sur leur en-tête (par
défaut, le nom croissant). Le pied de liste permet de changer de page et de
choisir le nombre de lignes par page.

La carte **Interrogation par la supervision** donne l'état du service SNMP de la Gateway, comme
l'écran :doc:`etat-des-services`, avec sa raison en cas d'erreur. Quand l'agent
est configuré, elle donne aussi :

- son adresse d'écoute et les versions qu'il accepte ;
- si une communauté est définie (*public, la valeur par défaut* quand c'est
  elle) ;
- l'utilisateur SNMPv3 et ses protocoles d'authentification et de chiffrement,
  avec la mention de la phrase secrète.

Sinon, elle indique que la Gateway ne répond pas aux requêtes SNMP.

Sans droit d'écriture, les boutons et les actions qui modifient la
configuration n'apparaissent pas ; le serveur refuse de toute façon une action
sans le droit correspondant.


La notification de test
=======================

Le bouton *Tester l'envoi à tous les moniteurs*, présent dès qu'un moniteur
existe, fait envoyer par la Gateway une notification d'erreur de transfert
fictive à tous les moniteurs. La page dit ensuite si l'envoi a réussi ou, s'il a
échoué, la cause qu'en donne le service SNMP.


Fiche d'un moniteur
===================

La fiche d'un moniteur, adresse ``/webui/snmp/monitors/<id>``, montre sa version
et son adresse, puis sa sécurité : la communauté en SNMPv2c ; en SNMPv3, le
niveau, l'utilisateur et, selon le niveau, l'identifiant du moteur
d'authentification et les protocoles, ainsi que le contexte s'il est renseigné.
Les boutons *Modifier le moniteur* et « ⋯ » (supprimer) sont sur la fiche ;
*Retour à SNMP* ramène à la page telle qu'on l'a quittée.


Créer ou modifier un moniteur
=============================

Le bouton *Ajouter un moniteur* ouvre le formulaire de création ; *Modifier le
moniteur* ouvre le même formulaire, pré-rempli.

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - Le nom (unique), la version et l'adresse UDP de l'outil de supervision,
       sous la forme ``hôte:port`` (le port des notifications est
       habituellement 162). Le choix de la version affiche le bloc qui lui
       correspond.
   * - SNMPv2c
     - La communauté ; vide, la Gateway emploie ``public``.
   * - SNMPv3
     - Le niveau de sécurité et l'utilisateur, obligatoire même sans
       authentification. Selon le niveau : l'identifiant du moteur
       d'authentification, le protocole et la phrase secrète
       d'authentification ; le protocole et la phrase secrète de chiffrement.
   * - Contexte SNMPv3
     - Bloc replié : le nom et l'identifiant du moteur de contexte, à renseigner
       seulement si l'outil de supervision les demande.

L'**identifiant du moteur d'authentification** s'écrit en hexadécimal, préfixé
de ``0x`` (``0x80001f8880e9630000d61ff449``), ou en décimal ; le formulaire
refuse une autre forme, que la Gateway enverrait telle quelle au lieu des
octets qu'elle désigne.

Les listes de protocoles signalent les plus anciens, à n'employer qu'avec un
outil de supervision qui n'en connaît pas d'autre : *MD5* et *SHA* (SHA-1) pour
l'authentification, *DES* pour le chiffrement.

À la modification, une phrase secrète ou une communauté laissée vide conserve la
valeur en place. Abaisser le niveau de sécurité efface ce que le nouveau niveau
n'emploie plus.

*Valider* enregistre le moniteur avec une notification et ramène à la page SNMP
(création) ou à la fiche du moniteur (modification) ; le service SNMP relit
aussitôt la liste des moniteurs. Un formulaire refusé est réaffiché avec les
valeurs saisies, sauf les secrets, et, sous chaque champ fautif, ce qu'il faut
corriger : nom vide ou déjà pris, version absente, adresse qui n'a pas la forme
``hôte:port`` ou qui ne se résout pas, utilisateur, identifiant, protocole ou
phrase secrète manquant.


Supprimer un moniteur
=====================

La suppression se confirme sur une page qui rappelle le moniteur. Après la
suppression, il ne reçoit plus les notifications de la Gateway.


Configurer ou désactiver l'agent
================================

*Configurer l'agent* (ou *Modifier l'agent*) ouvre le formulaire de l'agent :

- l'adresse d'écoute UDP, sous la forme ``hôte:port`` (``0.0.0.0:161`` pour
  toutes les interfaces, sur le port habituel) ;
- la communauté ; vide, la Gateway emploie ``public`` ;
- la case *Accepter seulement SNMPv3*, qui fait refuser les requêtes SNMPv2c ;
- l'utilisateur SNMPv3, ses protocoles d'authentification et de chiffrement et
  leurs phrases secrètes. Le chiffrement demande l'authentification ;
  l'utilisateur est obligatoire dès qu'un protocole est choisi ou que seul
  SNMPv3 est accepté.

L'enregistrement redémarre le service SNMP de la Gateway sur la nouvelle
configuration. Si le service ne la prend pas en compte (une adresse d'écoute
déjà occupée, par exemple), la configuration reste enregistrée et la page le
dit avec la cause.

*Désactiver l'agent*, dans le menu « ⋯ » de la carte, se confirme sur une page :
la configuration de l'agent est supprimée, la Gateway ne répond plus aux
requêtes SNMP ; les notifications continuent.


Et par l'API
============

Les moniteurs et l'agent se gèrent aussi par l':doc:`API REST
</reference/rest/snmp/index>` (``/api/snmp``) et par la ligne de commande
(``waarp-gateway snmp``). Ce qui est enregistré par l'un s'affiche et se
modifie par les autres.

L'ancienne page *SNMP*, qui renvoyait à la ligne de commande, mène désormais à
cet écran.
