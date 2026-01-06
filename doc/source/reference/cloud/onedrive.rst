========
OneDrive
========

Waarp Gateway permet d'utiliser un drive OneDrive (Sharepoint) à la place du
disque dur local.

Configuration
-------------

Pour utiliser une instance cloud, celle-ci doit d'abord être créée et configurée.
Pour créer une instance cloud OneDrive, le type renseigné doit être ``onedrive``
ou ``sharepoint``.

Authentification
^^^^^^^^^^^^^^^^

Avant toute chose, il est impératif d'enregistrer Gateway dans les applications
autorisées sur votre portail Azure. Uns fois cela fait, il vous faut ajouter une
autorisation pour cette application dans l'API Graph. Référez-vous à `ce guide
<https://learn.microsoft.com/fr-fr/azure/api-management/credentials-how-to-azure-ad>`_
pour la marche à suivre.

.. important::

   Pour que Gateway puisse fonctionner avec OneDrive, l'application a besoin à
   minima de la permission ``Files.ReadWrite.All`` sur l'API Graph. Prenez bien
   garde à ajouter une permission de type **"autorisation d'application"** et non
   de type "autorisation déléguée". Une fois l'autorisation créée, n'oubliez pas
   également de donner le consentement d'administrateur si cela s'avère nécessaire.

L'accès à un répertoire OneDrive se fait au moyen de l'API Graph de Microsoft.
L'authentification à celle-ci se fait via un token OAuth. Ce token peut soit
être directement fourni à Gateway via l'option ``token`` (voir chapitre suivant),
ou alors Gateway peut être configurée pour requérir le token elle-même.

Pour que Gateway puisse requérir le token, les éléments suivants sont nécessaires :

- Un ID d'application (aussi appelé "ID de client" ou *client ID*) doit être
  fournis via le paramètre *key* de l'instance cloud.
- Un secret d'application (aussi appelé "secret client") doit être fournis via
  le paramètre *secret* de l'instance cloud.

Options
^^^^^^^

Les options de configuration suivantes sont disponibles pour OneDrive :

* **tenant**: *REQUIS* - L'ID de l'annuaire (aussi appellé "locataire" ou *tenant*)
  Azure à utiliser.
* **drive_id**: *REQUIS* - L'ID du drive OneDrive à utiliser.
* **drive_type**: Le type de drive. Les valeurs autorisées sont : ``personal``,
  ``business`` ou ``documentLibrary``. La valeur par défaut est ``business``.
* **region**: La région du drive. Less valeurs autorisées sont : ``global``,
  ``us``, ``de`` et ``cn``. La valeur par défaut est ``global``.
* **token**: Le token OAuth2 à utiliser pour l'authentification.

Exemple
-------

Prenons le cas de figure suivant :

- fichier: ``doc/waarp-gateway.pdf``
- ID de drive: ``b!GoXLzaJCDEqhqT5H107_ksbhUsDDwtJPlR5kTWhg2CxYuCODHjHUTJOI3xBLJIMh``
- Drive personnel
- Région globale

Dans un premier temps, l'instance cloud doit être définie. Dans cet exemple, nous
lui donnerons le nom "ex-onedrive".

La commande de création pour cette instance cloud est donc :

.. code-block:: shell

   waarp-gateway cloud add -n "ex-onedrive" -t "onedrive" -o "drive_id:b!GoXLzaJCDEqhqT5H107_ksbhUsDDwtJPlR5kTWhg2CxYuCODHjHUTJOI3xBLJIMh" -o "drive_type:personal" -o "region:global"

Par la suite, lors de mon transfert, le chemin du fichier devra donc ressembler à :

| ex-onedrive:doc/waarp-gateway.pdf

